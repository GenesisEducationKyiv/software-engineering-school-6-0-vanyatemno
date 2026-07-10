package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is the subset of pgx query methods the repositories rely on. Both
// *pgxpool.Pool and pgx.Tx satisfy it, so a repository can run either directly
// against the pool or inside an ambient transaction obtained via WithTx. This
// is what lets the saga orchestrator compose several repositories' writes into
// a single atomic transaction (T1) without changing their query code.
//
// Begin is included so code that opens its own transaction (e.g.
// repositories/subscription manager.go::Delete) keeps working unchanged whether
// db is the pool (a real transaction) or an ambient tx (a savepoint-backed
// nested transaction in pgx/v5).
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Transactor runs a unit of work inside a single database transaction. It is
// the general form of the hand-rolled Begin/Rollback/Commit in
// repositories/subscription manager.go::Delete, extracted so the saga
// orchestrator can wrap writes across several repositories atomically.
type Transactor struct {
	pool *pgxpool.Pool
}

func NewTransactor(pool *pgxpool.Pool) *Transactor {
	return &Transactor{pool: pool}
}

// RunInTx opens a transaction, invokes fn with it, and commits if fn returns
// nil — rolling back on any error (or panic). Repositories participate by being
// re-bound to tx via their WithTx method.
func (t *Transactor) RunInTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
