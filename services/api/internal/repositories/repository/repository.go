package repository

import (
	"se-school/internal/infrastructure/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db db.DBTX
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{db: pool}
}

// WithTx returns a copy of the repository bound to tx so its writes join the
// caller's transaction (used by the saga orchestrator's atomic create).
func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{db: tx}
}

const repositoryColumns = "id, created_at, updated_at, deleted_at, owner, name, version"
