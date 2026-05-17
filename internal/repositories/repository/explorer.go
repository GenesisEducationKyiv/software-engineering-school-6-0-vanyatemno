package repository

import (
	"context"
	"errors"
	"se-school/internal/models"
	"se-school/internal/repositories"

	"github.com/jackc/pgx/v5"
)

func scanRepository(row pgx.Row) (*models.Repository, error) {
	var r models.Repository
	if err := row.Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt, &r.Owner, &r.Name, &r.Version); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *Repository) GetByID(ctx context.Context, id uint) (*models.Repository, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+repositoryColumns+` FROM repositories WHERE id = $1 AND deleted_at IS NULL`,
		id,
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

// Find looks up a repository by the non-zero fields of the supplied model
// (owner + name). Mirrors the prior GORM struct-based WHERE behaviour.
func (r *Repository) Find(ctx context.Context, repo *models.Repository) (*models.Repository, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+repositoryColumns+` FROM repositories
		 WHERE owner = $1 AND name = $2 AND deleted_at IS NULL`,
		repo.Owner, repo.Name,
	)
	found, err := scanRepository(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return found, nil
}

func (r *Repository) GetAll(ctx context.Context) ([]*models.Repository, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+repositoryColumns+` FROM repositories WHERE deleted_at IS NULL`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Repository
	for rows.Next() {
		repo, err := scanRepository(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, repo)
	}
	return result, rows.Err()
}
