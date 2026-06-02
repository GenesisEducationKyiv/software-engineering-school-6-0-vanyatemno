package repository

import (
	"context"
	"errors"
	"se-school/internal/models"
	"se-school/internal/repositories"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) Create(ctx context.Context, repository *models.Repository) error {
	row := r.db.QueryRow(ctx,
		`INSERT INTO repositories (owner, name, version)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		repository.Owner, repository.Name, repository.Version,
	)
	return row.Scan(&repository.ID, &repository.CreatedAt, &repository.UpdatedAt)
}

func (r *Repository) UpdateTag(ctx context.Context, id uint, tag string) (*models.Repository, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE repositories
		 SET version = $1, updated_at = NOW()
		 WHERE id = $2 AND deleted_at IS NULL
		 RETURNING `+repositoryColumns,
		tag, id,
	)
	repo, err := scanRepository(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return repo, nil
}

func (r *Repository) Delete(ctx context.Context, repository *models.Repository) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE repositories SET deleted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`,
		repository.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return repositories.ErrNotFound
	}
	return nil
}

// FindOrCreate returns the existing row matching (owner, name) or inserts a
// new row with the supplied values. The struct is mutated in place to reflect
// the persisted state (ID, timestamps, version).
func (r *Repository) FindOrCreate(ctx context.Context, repository *models.Repository) (*models.Repository, error) {
	found, err := r.Find(ctx, repository)
	if err == nil {
		return found, nil
	}
	if !errors.Is(err, repositories.ErrNotFound) {
		return nil, err
	}

	if err := r.Create(ctx, repository); err != nil {
		return nil, err
	}
	return repository, nil
}
