// Package outbox is the data-access layer for the transactional outbox. Rows are
// written inside the orchestrator's atomic create (via WithTx) and later drained
// by the relay, which publishes each to Kafka and marks it published.
package outbox

import (
	"context"

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

// WithTx returns a copy of the repository bound to tx so the command row is
// written in the same transaction as the subscription + saga rows.
func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{db: tx}
}

// Create persists an unpublished outbox message (id set by the caller).
func (r *Repository) Create(ctx context.Context, m *models.OutboxMessage) error {
	row := r.db.QueryRow(ctx,
		`INSERT INTO outbox (id, saga_id, topic, kafka_key, payload)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING created_at`,
		m.ID, m.SagaID, m.Topic, m.KafkaKey, m.Payload,
	)
	return row.Scan(&m.CreatedAt)
}

// GetUnpublished returns the oldest messages not yet published, up to limit.
func (r *Repository) GetUnpublished(ctx context.Context, limit int) ([]*models.OutboxMessage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, saga_id, topic, kafka_key, payload, published_at, attempts, last_error, created_at
		 FROM outbox
		 WHERE published_at IS NULL
		 ORDER BY created_at
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.OutboxMessage
	for rows.Next() {
		var m models.OutboxMessage
		var sagaID *string
		if err := rows.Scan(
			&m.ID, &sagaID, &m.Topic, &m.KafkaKey, &m.Payload,
			&m.PublishedAt, &m.Attempts, &m.LastError, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		if sagaID != nil {
			m.SagaID = *sagaID
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// MarkPublished stamps published_at, so the relay never re-publishes the row.
func (r *Repository) MarkPublished(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE outbox SET published_at = NOW() WHERE id = $1`, id)
	return err
}

// MarkFailed records a failed publish attempt (relay retries on the next tick).
func (r *Repository) MarkFailed(ctx context.Context, id, cause string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE outbox SET attempts = attempts + 1, last_error = $1 WHERE id = $2`,
		cause, id,
	)
	return err
}
