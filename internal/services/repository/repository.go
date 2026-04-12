package repository

import (
	"se-school/internal/integrations/github"
	"se-school/internal/notifications"
	"se-school/internal/repositories/repository"
	"se-school/internal/repositories/subscription"
)

type Service struct {
	repositoriesRepository  repository.RepositoriesRepository
	subscriptionsRepository subscription.SubscriptionsRepository
	notificationsService    notifications.NotificationsService
	githubService           github.GithubIntegration
}

func New(
	repositoriesRepository repository.RepositoriesRepository,
	subscriptionsRepository subscription.SubscriptionsRepository,
	notificationsService notifications.NotificationsService,
	githubService github.GithubIntegration,
) *Service {
	return &Service{
		repositoriesRepository:  repositoriesRepository,
		subscriptionsRepository: subscriptionsRepository,
		notificationsService:    notificationsService,
		githubService:           githubService,
	}
}
