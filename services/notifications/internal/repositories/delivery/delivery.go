// Package delivery is the data-access layer for the notifier's deliveries table
// — its durable participant state in the distributed transaction. Claim records
// (idempotently) that a confirmation email is being dispatched; SetState records
// the terminal outcome the saga reply is built from.
package delivery

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// State is the lifecycle of a single email dispatch.
type State = string

const (
	StateSending State = "SENDING"
	StateSent    State = "SENT"
	StateFailed  State = "FAILED"
)

// Delivery is one recipient's dispatch record for a saga command.
type Delivery struct {
	ID             int64
	SagaID         string
	IdempotencyKey string
	Recipient      string
	Template       string
	State          State
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Claim ensures a delivery row exists for (idempotency_key, recipient) and
// returns its current state. On a fresh insert it creates the row in SENDING and
// returns SENDING (the caller should send). On a redelivery it returns the
// existing state without change — SENT means already delivered (skip), while
// SENDING/FAILED mean the caller should (re)attempt the send. d.ID is populated
// so the caller can transition the row with SetState.
func (r *Repository) Claim(ctx context.Context, d *Delivery) (State, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO deliveries (saga_id, idempotency_key, recipient, template, state)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (idempotency_key, recipient)
		   DO UPDATE SET updated_at = NOW()
		 RETURNING id, state`,
		d.SagaID, d.IdempotencyKey, d.Recipient, d.Template, StateSending,
	)
	var id int64
	var state State
	if err := row.Scan(&id, &state); err != nil {
		return "", err
	}
	d.ID = id
	d.State = state
	return state, nil
}

// SetState transitions a delivery to its terminal outcome (SENT or FAILED),
// recording lastErr (empty when none).
func (r *Repository) SetState(ctx context.Context, id int64, state State, lastErr string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE deliveries SET state = $1, last_error = $2, updated_at = NOW() WHERE id = $3`,
		state, lastErr, id,
	)
	return err
}
