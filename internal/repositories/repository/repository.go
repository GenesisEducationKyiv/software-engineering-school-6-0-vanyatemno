package repository

import "github.com/jackc/pgx/v5/pgxpool"

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const repositoryColumns = "id, created_at, updated_at, deleted_at, owner, name, version"
