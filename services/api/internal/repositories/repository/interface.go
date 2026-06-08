package repository

import (
	"context"
	"se-school/internal/models"
)

type RepositoriesRepository interface {
	GetByID(ctx context.Context, id uint) (*models.Repository, error)
	GetAll(ctx context.Context) ([]*models.Repository, error)
	Find(ctx context.Context, repo *models.Repository) (*models.Repository, error)
	Create(ctx context.Context, repo *models.Repository) error
	FindOrCreate(ctx context.Context, repo *models.Repository) (*models.Repository, error)
	UpdateTag(ctx context.Context, id uint, tag string) (*models.Repository, error)
	Delete(ctx context.Context, repo *models.Repository) error
}
