package repository

type Service struct {
	frontendURL             string
	repositoriesRepository  RepositoriesRepository
	subscriptionsRepository SubscriptionsRepository
	notificationsService    NotificationsService
	githubService           GithubIntegration
}

func New(
	frontendURL string,
	repositoriesRepository RepositoriesRepository,
	subscriptionsRepository SubscriptionsRepository,
	notificationsService NotificationsService,
	githubService GithubIntegration,
) *Service {
	return &Service{
		frontendURL:             frontendURL,
		repositoriesRepository:  repositoriesRepository,
		subscriptionsRepository: subscriptionsRepository,
		notificationsService:    notificationsService,
		githubService:           githubService,
	}
}
