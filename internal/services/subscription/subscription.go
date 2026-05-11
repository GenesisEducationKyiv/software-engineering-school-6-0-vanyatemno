package subscription

type Service struct {
	frontendURL             string
	subscriptionsRepository SubscriptionsRepository
	repositoriesRepository  RepositoriesRepository
	codesRepository         CodesRepository
	githubIntegration       GithubIntegration
	notificationService     NotificationsService
}

func New(
	frontendURL string,
	subscriptionsRepository SubscriptionsRepository,
	repositoriesRepository RepositoriesRepository,
	codesRepository CodesRepository,
	githubIntegration GithubIntegration,
	notificationService NotificationsService,
) *Service {
	return &Service{
		frontendURL:             frontendURL,
		subscriptionsRepository: subscriptionsRepository,
		repositoriesRepository:  repositoriesRepository,
		codesRepository:         codesRepository,
		githubIntegration:       githubIntegration,
		notificationService:     notificationService,
	}
}
