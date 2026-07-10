# Application Architecture

UML views of the **GitHub Release Notification** system, rendered in Mermaid so they
display inline on GitHub. This document reflects the **current code**; where the older
`services/api/docs/system-design.md` diverges (e.g. it shows release alerts over Kafka),
this file is the source of truth.

The system lets a user subscribe (by email) to a public GitHub repository and receive an
email when that repo publishes a new release. It is split into **two independently
deployable Go microservices** plus a shared wire-contract module:

- **`api`** (`services/api`, module `se-school`) — Gin HTTP API + hourly release-poll cron
  + the orchestrator of the subscribe **saga**.
- **`notifier`** (`services/notifications`, module `ghnotify/notifier`) — email delivery
  over **two transports** (a Kafka consumer and a gRPC server) that share one at-most-once
  dispatch core.
- **`pkg/contract`** (module `ghnotify/contract`) — the pure wire types both sides agree on
  (Kafka envelopes + the gRPC `.proto`).

The two services **share no database**. They talk over two transports:

- **Kafka (async)** — the subscribe/confirmation **saga**: outbox → relay → worker →
  replies → sweeper.
- **gRPC (sync)** — repository-update **alerts**: the cron calls `Notifier.Notify` once per
  subscriber and only advances that subscriber's `last_seen_tag` when the call returns
  `ok = true`.

---

## 1. Component / Container view

Structural view (C4-container style): the two services, their internal components, the
infrastructure they depend on, and the protocol on every connector.

```mermaid
flowchart LR
    Client["Client / Browser"]
    GH["GitHub REST API"]
    SMTP["SMTP server"]

    subgraph api["api service · module se-school"]
        H["Gin HTTP handlers<br/>(controllers)"]
        SUB["Subscription service<br/>(saga orchestrator)"]
        REPO["Repository service"]
        CRON["Cron scheduler<br/>(robfig/cron, hourly)"]
        RELAY["Outbox relay"]
        RC["Reply consumer"]
        SW["Saga sweeper"]
        GHI["GitHub integration"]
        GC["gRPC client"]
    end

    subgraph notif["notifier service · module ghnotify/notifier"]
        WK["Kafka worker"]
        GS["gRPC server<br/>(Notifier.Notify)"]
        DP["Dispatch core<br/>(at-most-once)"]
        TPL["Template renderer"]
        MAIL["SMTP mailer"]
        DD["Dedup"]
    end

    PGA[("PostgreSQL<br/>gh-subscriptions")]
    PGN[("PostgreSQL<br/>gh-notifier")]
    RED[("Redis<br/>GitHub cache + dedup")]
    KAFKA[["Kafka<br/>events · .dlq · replies"]]

    Client -->|"HTTP + X-API-Key"| H
    H --> SUB
    CRON --> REPO
    SUB -->|"T1 (uow): sub + codes<br/>+ saga + outbox"| PGA
    REPO --> PGA
    REPO --> GHI
    GHI -->|"REST"| GH
    GHI -->|"cache"| RED
    REPO --> GC

    RELAY -->|"drain outbox"| PGA
    RELAY -->|"produce → notifications.events"| KAFKA
    RC -->|"consume → notifications.replies"| KAFKA
    RC --> PGA
    SW -->|"compensate past deadline"| PGA

    GC ==>|"gRPC Notify (sync alerts)"| GS
    KAFKA -->|"consume events"| WK

    WK --> DP
    GS --> DP
    DP --> TPL
    DP -->|"record deliveries"| PGN
    DP --> DD
    DP --> MAIL
    DD -->|"idempotency marker"| RED
    MAIL -->|"SMTP"| SMTP
    WK -->|"produce replies / dlq"| KAFKA
```

| Component | Role |
|---|---|
| **Gin HTTP handlers** | Serve `/api/*` (subscribe, status, confirm, unsubscribe, list); X-API-Key auth, request-id, logging, metrics middleware. |
| **Subscription service** | Orchestrates the subscribe saga; the `uow` atomic write (T1) and the `HandleReply` / `Sweep` state transitions. |
| **Repository service** | Cron target: polls GitHub for new tags and fans out release alerts over the gRPC client. |
| **Cron scheduler** | Fires `CheckAllReposTagAndAlert` on `CRON_REPO_CHECK_SCHEDULE` (default hourly). |
| **Outbox relay / Reply consumer / Saga sweeper** | Background loops: drain the outbox → Kafka; consume saga replies; compensate sagas whose reply never arrives before `deadline_at`. |
| **GitHub integration** | `go-github` client; latest-release lookups cached in Redis (10 min TTL). |
| **gRPC client → gRPC server** | Synchronous `Notifier.Notify` path for repository-update alerts (`notifier:9090`). |
| **Kafka worker** | Consumes `notifications.events` (saga commands), publishes replies / dead-letters. |
| **Dispatch core** | Shared at-most-once delivery: claim `deliveries` row → dedup → render → SMTP with retry; `SENDING → SENT/FAILED`. |
| **PostgreSQL ×2** | `gh-subscriptions` (API: repositories, subscriptions, codes, saga_instances, outbox) and `gh-notifier` (notifier: deliveries). No shared DB. |
| **Redis** | Dual use: GitHub release-lookup cache (API) **and** per-recipient idempotency store (notifier). |
| **Kafka** | Durable transport: `notifications.events`, `notifications.events.dlq`, `notifications.replies`. |

---

## 2. Behavioral views (sequence diagrams)

### 2a. Subscribe — orchestrated saga (async, Kafka)

Distributed transaction across the two databases. Invariant: a subscription is durable
(awaiting the confirm click) **iff** its confirmation email was dispatched; otherwise it is
compensated away.

```mermaid
sequenceDiagram
    autonumber
    actor C as Client
    participant API as API (orchestrator)
    participant PgA as Postgres (gh-subscriptions)
    participant K as Kafka
    participant N as Notifier
    participant PgN as Postgres (gh-notifier)
    participant M as SMTP

    C->>API: POST /api/subscribe {email, repo}
    Note over API,PgA: resolve/create repository<br/>(GitHub, Redis-cached) before T1
    Note over API,PgA: T1 — one tx (uow): subscription + confirm/unsubscribe<br/>codes + saga(AWAITING_NOTIFICATION) + outbox command
    API-->>C: 202 Accepted {sagaId, state}

    loop relay tick
        API->>PgA: read unpublished outbox
        API->>K: produce contract.Message → notifications.events
    end

    K->>N: consume command (sagaId, template=confirm)
    Note over N,PgN: deliveries SENDING → render → send → SENT/FAILED
    N->>M: send confirmation email
    N->>K: produce contract.Reply → notifications.replies

    K->>API: consume reply (sagaId, status)
    alt status = dispatched
        API->>PgA: saga → COMPLETED
    else status = failed
        API->>PgA: compensate (soft-delete sub + codes) → COMPENSATED
    end

    Note over API,PgA: sweeper: sagas past deadline_at → compensate (reply lost / notifier down)

    C->>API: GET /api/subscribe/status/{sagaId}
    API-->>C: saga state
```

### 2b. Release poll & alert — cron → synchronous gRPC

The hourly cron detects new release tags and notifies lagging subscribers one at a time over
gRPC. `last_seen_tag` advances **only** on a confirmed (`ok=true`) delivery, so a failed
recipient is retried next run without blocking the others.

```mermaid
sequenceDiagram
    autonumber
    participant Cron as Cron (hourly)
    participant PgA as Postgres (gh-subscriptions)
    participant RED as Redis
    participant GH as GitHub API
    participant N as gRPC Notifier
    participant M as SMTP

    Cron->>PgA: GetAll repositories
    loop each repository
        Cron->>RED: GET github:repo_version:{owner}/{repo}
        alt cache miss
            Cron->>GH: GET latest release tag
            GH-->>Cron: tag
            Cron->>RED: SET tag (TTL 10m)
        end
        opt tag changed
            Cron->>PgA: UpdateTag(repositories.version)
        end
        Cron->>PgA: GetUnupdated(repo, currentVersion)
        loop each lagging subscriber
            Cron->>N: gRPC Notify(template=repository_update, recipient, idempotencyKey)
            Note over N,M: dispatch: dedup → render → SMTP<br/>deliveries SENDING → SENT/FAILED
            N-->>Cron: NotifyResponse {ok, reason}
            alt ok = true
                Cron->>PgA: UpdateLastSeenTag(sub, currentVersion)
            else ok = false
                Note over Cron: leave last_seen_tag → retried next run
            end
        end
    end
```

### 2c. Confirm subscription

```mermaid
sequenceDiagram
    autonumber
    actor C as Client
    participant API as API
    participant PgA as Postgres (gh-subscriptions)

    C->>API: GET /api/confirm/{token}
    API->>PgA: find code (type=confirmation)
    alt invalid / expired
        API-->>C: 4xx error
    else
        API->>PgA: subscription.is_confirmed = true
        API->>PgA: delete confirmation code
        API-->>C: 200 OK
    end
```

### 2d. Unsubscribe

```mermaid
sequenceDiagram
    autonumber
    actor C as Client
    participant API as API
    participant PgA as Postgres (gh-subscriptions)

    C->>API: GET /api/unsubscribe/{token}
    API->>PgA: find code (type=unsubscribe)
    API->>PgA: soft-delete subscription (deleted_at)
    API-->>C: 200 OK
```

---

## References (source of truth in code)

| Concern | File(s) |
|---|---|
| gRPC contract (`Notifier.Notify`, `NotifyRequest/Response`) | `pkg/contract/notifierpb/notifier.proto` |
| Kafka wire types, topics, template names | `pkg/contract/contract.go` |
| gRPC server / client | `services/notifications/internal/grpcserver/server.go`, `services/api/internal/notifications/grpcclient/` |
| Cron alert path (sync gRPC, `ok`-gated tag advance) | `services/api/internal/services/repository/update.go` |
| Saga orchestrator (outbox, relay, replies, sweeper, uow) | `services/api/internal/services/subscription/saga.go`, `services/api/internal/notifications/{relay,replies}/`, `services/api/internal/uow/uow.go` |
| Delivery core (at-most-once) | `services/notifications/internal/dispatch/dispatch.go` |
| HTTP routes | `services/api/internal/controllers/router.go` |
| Service entrypoints | `services/api/cmd/main.go`, `services/notifications/cmd/main.go` |
| Saga decision record | `services/api/docs/adrs/ADR-005-orchestrated-saga-decision.md` |
