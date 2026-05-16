package repository

import (
	"context"
	"se-school/internal/models"
)

type RepositoriesService interface {
	CheckRepoTagAndAlert(ctx context.Context, repo *models.Repository) error
	CheckAllReposTagAndAlert(ctx context.Context) error
}
