# ADR-004 Database access library decision
* Status: accepted
* Date: 2026-05-17
* Author: Ivan Markhaichuk

## Context
ADR-001 chose PostgreSQL as the storage engine but did not pin a specific data-access library.
The project currently uses GORM, which served well to bootstrap the schema and CRUD layer, but several friction points have become visible as the codebase matures:
* GORM hides the SQL it generates, which makes performance analysis and debugging (N+1s, unexpected joins) harder.
* Reflection-based query building adds runtime overhead on every call.
* PostgreSQL-specific features (partial unique indexes, `ON CONFLICT`, JSONB operators, `COPY`) require escape hatches or raw SQL anyway.
* `gorm.Model` and the `AfterDelete` hook described in ADR-001 introduce implicit behaviour (soft-delete filtering, non-transactional cascades) that is invisible at the call site.

We need to decide whether to keep GORM or move to a lower-level library.

---

## Considered technologies
1. GORM (`gorm.io/gorm` + `gorm.io/driver/postgres`) — status quo
    1. Pros: terse CRUD API, built-in `AutoMigrate`, model hooks, soft-delete handling, struct-tag-driven schema
    2. Cons: hidden generated SQL, reflection overhead on every query, PostgreSQL features require raw-SQL escape hatches, hooks run outside explicit transactions, implicit `deleted_at` filtering surprises new readers
2. pgx (`github.com/jackc/pgx/v5` + `pgxpool`)
    1. Pros: native PostgreSQL driver (binary wire protocol), explicit SQL at every call site, first-class support for PG-specific types and features, fine-grained pool control via `pgxpool`, typed PG error codes already used in `internal/infrastructure/db/errors.go`
    2. Cons: more boilerplate per repository method, `AutoMigrate` must be replaced by a dedicated migration tool, cascade behaviour must be expressed explicitly

---

## Chosen library: `pgx` (with `pgxpool`)
The data-access surface of this project is small — roughly twenty query sites split across the `explorer`/`manager` repositories — so the additional boilerplate cost of explicit SQL is bounded, while the benefits (predictable queries, performance, direct access to PostgreSQL features) apply to every call.
pgx is already a transitive dependency of the current `gorm.io/driver/postgres`, and the existing duplicate-key check in `internal/infrastructure/db/errors.go` already relies on pgx error types — promoting pgx to a direct dependency does not enlarge the dependency footprint in a meaningful way.

**Notes:**
- The schema described in ADR-001 is unchanged. The `id`, `created_at`, `updated_at`, `deleted_at` columns remain; only the `gorm.Model` embedding in Go is removed.
- Soft deletes become explicit: `WHERE deleted_at IS NULL` on every read, `UPDATE ... SET deleted_at = NOW()` on every delete.
- `AutoMigrate` is replaced by a dedicated migration tool (e.g. `golang-migrate`); the partial unique index on `subscriptions(email, repository_id) WHERE deleted_at IS NULL` moves into a versioned migration file.
- The `AfterDelete` cascade on `subscriptions` moves into the service layer and runs inside a `pgx.Tx`, making the cascade atomic — an improvement over the current non-transactional hook.
- Repository interfaces stay the same; only the GORM-backed implementations are rewritten. Service-layer code and mock-based tests are unaffected.

## Consequences
### Positive
* Every database interaction is visible as plain SQL — no hidden N+1s, no implicit filters
* Lower per-query overhead and memory footprint (no reflection, binary protocol)
* Direct access to PostgreSQL features (`ON CONFLICT`, partial indexes, JSONB, `COPY`, `LISTEN/NOTIFY`) without escape hatches
* Cascade deletes become transactional and explicit
* Removes two dependencies (`gorm.io/gorm`, `gorm.io/driver/postgres`)
### Negative
* More boilerplate per repository method (manual row scanning, explicit column lists)
* Schema changes now require hand-written migration files instead of struct-tag edits
* Soft-delete filtering must be remembered at every read site
* Locks the project to PostgreSQL at the code level (acceptable — ADR-001 already committed to PostgreSQL)
