package subscription

import "github.com/jackc/pgx/v5/pgxpool"

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const subscriptionColumns = `
	id, created_at, updated_at, deleted_at,
	repository_id, subscribe_code_id, unsubscribe_code_id,
	email, is_confirmed, last_seen_tag`
