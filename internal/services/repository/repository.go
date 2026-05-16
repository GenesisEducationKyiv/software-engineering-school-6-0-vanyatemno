package repository

import (
	"se-school/internal/integrations/github"
	"se-school/internal/notifications"
	"se-school/internal/repositories/repository"
	"se-school/internal/repositories/subscription"
)

type Service struct {
	frontendURL             string
	repositoriesRepository  repository.RepositoriesRepository
	subscriptionsRepository subscription.SubscriptionsRepository
	notificationsService    notifications.NotificationsService
	githubService           github.GithubIntegration
}

func New(
	frontendURL string,
	repositoriesRepository repository.RepositoriesRepository,
	subscriptionsRepository subscription.SubscriptionsRepository,
	notificationsService notifications.NotificationsService,
	githubService github.GithubIntegration,
) *Service {
	return &Service{
		frontendURL:             frontendURL,
		repositoriesRepository:  repositoriesRepository,
		subscriptionsRepository: subscriptionsRepository,
		notificationsService:    notificationsService,
		githubService:           githubService,
	}
}
