package cron

import (
	"context"
	"se-school/internal/models"
)

// CronScheduler defines the interface for managing cron jobs.
type CronScheduler interface {
	Start()
	Stop()
}

type RepositoriesService interface {
	CheckRepoTagAndAlert(ctx context.Context, repo *models.Repository) error
	CheckAllReposTagAndAlert(ctx context.Context) error
}
