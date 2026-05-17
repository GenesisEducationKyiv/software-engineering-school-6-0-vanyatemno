# Testing

Prerequisites on the host:

- `git`
- `docker` (with the `docker compose` plugin)
- `go` 1.26 — only required for unit tests; integration tests run Go inside a container

Clone, then run the command for the test type you want. Each command is self-contained — no extra setup, no manual DB/Redis provisioning.

## Unit tests

```bash
make test-unit
```

What it runs: `go test -race -count=1 ./internal/...` on the host. Uses in-package mocks (`internal/**/mock.go`), no external services. Race detector on. Test cache disabled (`-count=1`) so reruns always re-execute.

Scope: every package under `internal/` that has `*_test.go` (services, repositories, integrations, utils, templates).

## Integration tests

```bash
make test-integration
```

What it runs:

1. `docker compose -f docker-compose.test.yml up --build --abort-on-container-exit --exit-code-from tests`
   - Builds the test runner image from `Dockerfile.test` (Go 1.26 alpine).
   - Starts disposable `postgres:16.3-alpine` and `redis:7-alpine` containers with healthchecks; the test runner waits on `service_healthy` before starting.
   - Runs `go test -count=1 -v ./tests/integration/...` inside the runner container against the in-network DB + Redis.
   - GitHub API calls are stubbed by an in-process MSW-style mock (`tests/integration/helpers/mswgh.go`) — no outbound network.
2. `docker compose -f docker-compose.test.yml down -v` tears everything down, including the named volumes, so the next run starts from a clean state.

Configuration lives in `.env.test` (committed; DB DSN + Redis address point at the compose-internal hostnames).

Teardown only, if a previous run crashed mid-flight:

```bash
make test-integration-down
```

## End-to-end tests

```bash
make test-e2e
```

What it runs:

1. `docker compose --env-file .env.e2e -f docker-compose.e2e.yml up --build --abort-on-container-exit --exit-code-from tests` brings up the full stack:
   - `postgres:16.3-alpine` + `redis:7-alpine` (tmpfs, healthchecked).
   - `axllent/mailpit` as the SMTP target so tests can assert email delivery and pull confirmation tokens out of the rendered HTML.
   - The backend, built from `Dockerfile`, pointed at the in-network Postgres/Redis/Mailpit and the **real** GitHub API. The release-check cron is parked at `0 0 1 1 *` so it doesn't fire during a run.
   - The frontend (`tests/e2e/Dockerfile.frontend`) clones [`vanyatemno/se-school-2026-frontend`](https://github.com/vanyatemno/se-school-2026-frontend), builds it with `VITE_API_URL=http://backend:8080/api` / `VITE_API_KEY=e2e-key`, and serves the static bundle from nginx on `:4173` with SPA fallback.
   - The `tests` runner (`tests/e2e/Dockerfile.runner`) installs Playwright browsers + OS deps and runs `go test ./tests/e2e/...` against the stack.
2. `docker compose ... down -v` tears everything down.

The Go test module lives at `tests/e2e/go.mod` (separate from the main module so the Playwright/pgx deps don't bleed into the application).

**Prerequisite:** copy the example file and fill in a token:

```bash
cp .env.e2e.example .env.e2e
# then edit .env.e2e and set GITHUB_TOKEN=...
```

The token is a GitHub personal access token (classic, `public_repo` scope). The backend hits the live GitHub API to validate repositories during subscribe — no GitHub stub is used. `.env.e2e` is gitignored.

Teardown only, if a previous run crashed mid-flight:

```bash
make test-e2e-down
```

## CI

GitHub Actions runs each suite in its own workflow on every push to `main` and every PR:

- `.github/workflows/unit-tests.yml` → `make test-unit`
- `.github/workflows/integration-tests.yml` → `make test-integration`
