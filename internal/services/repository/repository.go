package repository

import (
	"se-school/internal/config"
	"se-school/internal/integrations/github"
	"se-school/internal/notifications"
	"se-school/internal/repositories/repository"
	"se-school/internal/repositories/subscription"
)

type Service struct {
	cfg                     *config.Config
	repositoriesRepository  repository.RepositoriesRepository
	subscriptionsRepository subscription.SubscriptionsRepository
	notificationsService    notifications.NotificationsService
	githubService           github.GithubIntegration
}

func New(
	cfg *config.Config,
	repositoriesRepository repository.RepositoriesRepository,
	subscriptionsRepository subscription.SubscriptionsRepository,
	notificationsService notifications.NotificationsService,
	githubService github.GithubIntegration,
) *Service {
	return &Service{
		cfg:                     cfg,
		repositoriesRepository:  repositoriesRepository,
		subscriptionsRepository: subscriptionsRepository,
		notificationsService:    notificationsService,
		githubService:           githubService,
	}
}
