.# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Go 1.26 REST API. Users subscribe an email to a GitHub repo (`owner/repo`); a cron job polls GitHub for new releases and emails confirmed subscribers. Module path: `se-school`.

## Commands

| Task | Command |
|---|---|
| Run locally | `go run cmd/main.go` (reads `.env` via Viper; needs Postgres + Redis up) |
| Run full stack | `docker compose up --build` (Postgres 16 + Redis 7 + backend) |
| Deps | `make dependencies` (= `go mod tidy && go mod download`) |
| Lint | `make lint` (golangci-lint v2, auto-installs to `$(go env GOPATH)/bin`) |
| Regen Swagger | `make swagger` — required after editing handler `@Summary/@Param/@Router` annotations; output goes to `docs/generated/` and is blank-imported by `cmd/main.go` |
| Tests | `go test ./...` |
| Single test | `go test ./internal/services/subscription -run TestName -v` |
| Install git hooks | `go run github.com/evilmartians/lefthook/v2 install` (pre-commit runs `make lint` + `make swagger` in parallel) |

`.env` is required; copy `.env.example`. Config keys are nested: env `DB_DSN` → `cfg.Database.DSN`, `MAILER_PORT` → `cfg.Mailer.Port`, etc. (Viper maps via `mapstructure` tags in `internal/config/config.go`.)

## Architecture

Layered, wired together in `cmd/main.go`:

```
controllers (Gin) → services → repositories (GORM) → Postgres
                  → integrations/github (cached in Redis)
                  → notifications → mailer (gomail/SMTP) + templates
cron/Scheduler → repositoryService.CheckAllReposTagAndAlert
```

All cross-layer deps are passed as **interfaces** (see `*/interface.go` and `services/subscription/interfaces.go`). Each package that exposes an interface also ships a hand-rolled `mock.go` next to it — use those in tests, not a generator.

### Routing & middleware (`internal/controllers/router.go`)

- `CORSMiddleware` + `PrometheusMiddleware` are global.
- `/swagger/*any` and `/metrics` sit outside the protected group.
- Everything under `/api` goes through `ErrorHandlerMiddleware` then `APIKeyMiddleware` (header `X-API-Key`; disabled when `SERVER_API_KEY` empty).

### Error handling convention

Handlers do **not** write error JSON directly. They call `c.Error(err)` and `return`. `ErrorHandlerMiddleware` (`controllers/middlewares/error_handler.go`) maps sentinel errors from `internal/models/errors.go` (`ErrNotFound`, `ErrAlreadyExists`, `ErrRepositoryNotFound`) to HTTP status. New domain errors must be added to that switch — otherwise they fall through to 500.

### Repository package layout

In `internal/repositories/{repository,subscription}/`, queries are split across two files by intent: **`manager.go`** (Create / Save / Update / Delete / FindOrCreate) and **`explorer.go`** (read-only Get*/Find). Keep this split when adding methods. `db.IsDuplicateKeyError` (in `internal/infrastructure/db/errors.go`) is used to translate Postgres unique-violation errors into `models.ErrAlreadyExists`.

### DB migrations

No migration tool. `internal/infrastructure/db/db.go` calls `Migrate(db)` on every model that implements `models.MigratableModel` (currently `Subscription`, `Repository`, `Code`). Each model's `Migrate` just calls `db.AutoMigrate(c)`. New tables must be registered in `db.Connect` and implement the interface.

### Code factory + policies

`internal/models/factories/codes/` produces confirmation (6-char, 30min TTL) and unsubscribe (UUID, 10y TTL) codes. To add a new `CodeType`, add a constant in `models/code.go` and an entry in the `policies` map (`policies.go`) — the factory looks up TTL + generator from there.

### GitHub integration

`internal/integrations/github/github.go` caches latest release tag in Redis under `github:repo_version:{owner}/{repo}` for 10 minutes. On GitHub rate-limit it **blocks** until reset (or ctx cancel) then retries. On 404 returns `models.ErrRepositoryNotFound`. Redis failures degrade gracefully — fall back to API, log warning.

### Cron

`internal/cron/cron.go` registers `CheckAllReposTagAndAlert` driven by `CRON_REPO_CHECK_SCHEDULE` (default hourly). The scheduler holds the app `context.Context` so jobs cancel on shutdown.

## Docs

- `docs/system-design.md` — system overview / sequence diagrams
- `docs/adrs/` — ADRs for DB / mail / GitHub integration choices
- `docs/source-swagger.yaml` — hand-written OpenAPI; **not** the source for the served UI (the generated dir is)

## Lint config

`.golangci.yml` uses `version: 2`, `default: none`, explicit allowlist (revive, staticcheck, unused, govet, misspell, etc.). Several SA1019 deprecation noises are silenced repo-wide — don't widen the allowlist without reason.
