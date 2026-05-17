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

## CI

GitHub Actions runs each suite in its own workflow on every push to `main` and every PR:

- `.github/workflows/unit-tests.yml` → `make test-unit`
- `.github/workflows/integration-tests.yml` → `make test-integration`
