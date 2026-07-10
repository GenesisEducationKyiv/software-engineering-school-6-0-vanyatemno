package cron

import (
	"context"
	"se-school/internal/models"
)

type CronScheduler interface {
	Start()
	Stop()
}

type RepositoriesService interface {
	CheckRepoTagAndAlert(ctx context.Context, repo *models.Repository) error
	CheckAllReposTagAndAlert(ctx context.Context) error
}
