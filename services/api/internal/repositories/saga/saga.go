// Package saga is the data-access layer for saga_instances, the durable state
// machine of the orchestrated Subscribe→confirmation-email saga.
package saga

import (
	"context"
	"errors"
	"time"

	"se-school/internal/infrastructure/db"
	"se-school/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sagaColumns = `id, type, state, subscription_id, email, idempotency_key,
	attempts, last_error, deadline_at, created_at, updated_at`

type Repository struct {
	db db.DBTX
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{db: pool}
}

// WithTx returns a copy of the repository bound to tx so its writes join the
// orchestrator's atomic create (T1).
func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{db: tx}
}

func scanSaga(row pgx.Row) (*models.SagaInstance, error) {
	var s models.SagaInstance
	if err := row.Scan(
		&s.ID, &s.Type, &s.State, &s.SubscriptionID, &s.Email,
		&s.IdempotencyKey, &s.Attempts, &s.LastError, &s.DeadlineAt,
		&s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) Create(ctx context.Context, s *models.SagaInstance) error {
	row := r.db.QueryRow(ctx,
		`INSERT INTO saga_instances
			(id, type, state, subscription_id, email, idempotency_key, attempts, last_error, deadline_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING created_at, updated_at`,
		s.ID, s.Type, s.State, s.SubscriptionID, s.Email,
		s.IdempotencyKey, s.Attempts, s.LastError, s.DeadlineAt,
	)
	return row.Scan(&s.CreatedAt, &s.UpdatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id string) (*models.SagaInstance, error) {
	row := r.db.QueryRow(ctx, `SELECT `+sagaColumns+` FROM saga_instances WHERE id = $1`, id)
	s, err := scanSaga(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	return s, nil
}

func (r *Repository) UpdateState(ctx context.Context, id string, state models.SagaState, lastErr string) error {
	res, err := r.db.Exec(ctx,
		`UPDATE saga_instances SET state = $1, last_error = $2, updated_at = NOW() WHERE id = $3`,
		state, lastErr, id,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *Repository) IncrementAttempts(ctx context.Context, id string) (int, error) {
	var attempts int
	row := r.db.QueryRow(ctx,
		`UPDATE saga_instances SET attempts = attempts + 1, updated_at = NOW() WHERE id = $1 RETURNING attempts`,
		id,
	)
	if err := row.Scan(&attempts); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, models.ErrNotFound
		}
		return 0, err
	}
	return attempts, nil
}

func (r *Repository) GetStuck(ctx context.Context, state models.SagaState, before time.Time, limit int) ([]*models.SagaInstance, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+sagaColumns+`
		 FROM saga_instances
		 WHERE state = $1 AND deadline_at < $2
		 ORDER BY deadline_at
		 LIMIT $3`,
		state, before, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.SagaInstance
	for rows.Next() {
		s, err := scanSaga(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
