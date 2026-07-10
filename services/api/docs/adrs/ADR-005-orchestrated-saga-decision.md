# ADR-005 Distributed transaction via an orchestrated saga

* Status: accepted
* Date: 2026-07-10
* Author: Ivan Markhaichuk

## Context

The **Subscribe** flow spans two microservices: the API service persists the
subscription in its PostgreSQL database, and the notifications service delivers
the confirmation email. Until now these were coupled by a single one-way Kafka
topic (`notifications.events`) and the API treated *"published to Kafka"* as
success. That left two consistency holes:

1. **Dual-write.** `Service.Create` committed the subscription + code rows, then
   published the email job in a *separate* step. A crash between the two left an
   orphaned, unconfirmable subscription with no email ever sent — and the
   hand-rolled "delete the subscription if the publish fails" compensation was
   best-effort and non-durable.
2. **No delivery outcome.** Communication was fire-and-forget: the API never
   learned whether the email was actually dispatched, so it could not react to a
   genuine delivery failure.

We need the two services to participate in one logical transaction — *a
subscription exists (awaiting the user's confirmation click) if and only if its
confirmation email was dispatched* — without a distributed lock or two-phase
commit across two databases.

---

## Considered approaches

1. **Two-phase commit (2PC / XA).** Atomic across both resources.
   * Cons: requires a coordinator and XA support across Postgres + Kafka + SMTP;
     blocking protocol, poor availability, not supported by the Kafka client or
     SMTP. Rejected outright for a message-driven system.
2. **Choreographed saga.** Each service reacts to the other's events with no
   central coordinator.
   * Pros: no coordinator; loose coupling.
   * Cons: the end-to-end transaction is implicit, smeared across event handlers;
     hard to see "what state is this subscription's transaction in?"; compensation
     logic is scattered.
3. **Orchestrated saga.** A central orchestrator (in the API) drives each step
   and runs compensations, backed by a persisted state machine.
   * Pros: the transaction is explicit and observable (one `saga_instances` row
     per transaction); compensation lives in one place; maps directly onto the
     existing "command → reply" shape and reuses the existing DLQ/retry/dedup.
   * Cons: the orchestrator is a (logical) coordinator that must itself be made
     durable and idempotent.

For emission atomicity we additionally considered publishing the command inline
(the status quo dual-write) versus a **transactional outbox** (persist the
command in the same DB transaction as the state change; a relay publishes it).

---

## Decision

Implement the Subscribe flow as an **orchestrated saga** with a **transactional
outbox**, making it a genuine two-database distributed transaction:

* **Orchestrator** lives in the API's subscription service. It persists a
  `saga_instances` state machine
  (`AWAITING_NOTIFICATION → COMPLETED | COMPENSATED | FAILED`).
* **T1 (API local transaction).** In one `pgx.Tx` (via a new `Transactor` /
  `UnitOfWork`) the orchestrator writes the subscription, its two codes, the saga
  instance **and** an `outbox` row carrying the confirmation-email command. State
  change and intent-to-send are now atomic — the dual-write is gone.
* **Outbox relay.** A background loop publishes unpublished outbox rows to
  `notifications.events` (at-least-once) and marks them published.
* **Notifier (participant local transaction).** The notifications service is
  given **its own PostgreSQL database** and a `deliveries` table. It records each
  dispatch (`SENDING → SENT/FAILED`) as its local commit — the durable
  at-most-once guard (surviving Redis-TTL expiry) — then publishes a **reply** on
  the new `notifications.replies` topic.
* **Reply-driven completion / compensation.** The API consumes replies:
  `dispatched → COMPLETED`; `failed → COMPENSATING →` delete the subscription +
  codes (reusing the existing atomic `Delete`) `→ COMPENSATED`.
* **Deadline sweeper.** A background loop compensates sagas whose reply never
  arrived, so a lost reply or a downed notifier cannot strand a subscription.
* **Async API.** `POST /api/subscribe` returns `202 Accepted` with a saga id;
  clients poll `GET /api/subscribe/status/:sagaId`.

Idempotency is anchored on the existing content-derived `contract.IdempotencyKey`
plus the Redis dedup store and the `deliveries` unique constraint, so re-published
commands and redelivered/duplicate replies are safe.

---

## Consequences

### Positive
* The create-then-publish dual-write is eliminated; a crash can no longer strand a subscription.
* The transaction is explicit and observable — one saga row per Subscribe, with a queryable state.
* The API now learns the real dispatch outcome and compensates deterministically.
* Reuses existing primitives (idempotency key, Redis dedup, DLQ, retry loop, the atomic `Delete`).
* No 2PC / distributed locks; each service commits only its own local transaction.

### Negative
* More moving parts in the API: an outbox relay, a reply consumer and a sweeper (three background loops), plus graceful shutdown.
* The notifications service — previously deliberately DB-less — now owns a database (a `deliveries` table) and its own migrations.
* Idempotency is enforced in two places for saga messages (Redis dedup + `deliveries` unique constraint); a future step could consolidate onto the table.
* Subscribe is now eventually consistent from the client's view (202 + polling) instead of a single synchronous 200.
* Known saga anomaly: if an email is sent but its reply is lost past the deadline, the sweeper compensates and the emailed confirm link 404s. Mitigated by a generous deadline; acceptable under saga semantics.
