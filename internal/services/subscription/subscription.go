package subscription

import (
	"se-school/internal/config"
	"se-school/internal/integrations/github"
	"se-school/internal/notifications"
	"se-school/internal/repositories/code"
	"se-school/internal/repositories/repository"
	"se-school/internal/repositories/subscription"
)

type Service struct {
	cfg                     *config.Config
	subscriptionsRepository subscription.SubscriptionsRepository
	repositoriesRepository  repository.RepositoriesRepository
	codesRepository         code.CodesRepository
	githubIntegration       github.GithubIntegration
	notificationService     notifications.NotificationsService
}

func New(
	cfg *config.Config,
	subscriptionsRepository subscription.SubscriptionsRepository,
	repositoriesRepository repository.RepositoriesRepository,
	codesRepository code.CodesRepository,
	githubIntegration github.GithubIntegration,
	notificationService notifications.NotificationsService,
) *Service {
	return &Service{
		cfg:                     cfg,
		subscriptionsRepository: subscriptionsRepository,
		repositoriesRepository:  repositoriesRepository,
		codesRepository:         codesRepository,
		githubIntegration:       githubIntegration,
		notificationService:     notificationService,
	}
}
