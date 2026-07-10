package code

import (
	"context"
	"errors"
	"se-school/internal/infrastructure/db"
	"se-school/internal/models"

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

const codeColumns = "id, created_at, updated_at, deleted_at, code, type, expires_at"

func scanCode(row pgx.Row) (*models.Code, error) {
	var c models.Code
	if err := row.Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt, &c.Code, &c.Type, &c.ExpiresAt); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) Get(ctx context.Context, codeString string) (*models.Code, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+codeColumns+` FROM codes WHERE code = $1 AND deleted_at IS NULL`,
		codeString,
	)
	c, err := scanCode(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

// Create persists a code that was already built by the codes factory. Generation and
// expiration live in the factory (SRP); the repository only handles data access.
func (r *Repository) Create(ctx context.Context, code *models.Code) error {
	row := r.db.QueryRow(ctx,
		`INSERT INTO codes (code, type, expires_at)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		code.Code, code.Type, code.ExpiresAt,
	)
	return row.Scan(&code.ID, &code.CreatedAt, &code.UpdatedAt)
}

func (r *Repository) Delete(ctx context.Context, id uint) error {
	_, err := r.db.Exec(ctx,
		`UPDATE codes SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	return err
}
