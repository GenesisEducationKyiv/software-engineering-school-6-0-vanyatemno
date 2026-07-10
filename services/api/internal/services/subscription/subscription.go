package subscription

import "time"

type Service struct {
	frontendURL             string
	subscriptionsRepository SubscriptionsRepository
	repositoriesRepository  RepositoriesRepository
	codesRepository         CodesRepository
	codeFactory             CodeFactory
	githubIntegration       GithubIntegration
	uow                     UnitOfWork
	sagaRepository          SagaRepository
	sagaDeadline            time.Duration
}

func New(
	frontendURL string,
	subscriptionsRepository SubscriptionsRepository,
	repositoriesRepository RepositoriesRepository,
	codesRepository CodesRepository,
	codeFactory CodeFactory,
	githubIntegration GithubIntegration,
	uow UnitOfWork,
	sagaRepository SagaRepository,
	sagaDeadline time.Duration,
) *Service {
	return &Service{
		frontendURL:             frontendURL,
		subscriptionsRepository: subscriptionsRepository,
		repositoriesRepository:  repositoriesRepository,
		codesRepository:         codesRepository,
		codeFactory:             codeFactory,
		githubIntegration:       githubIntegration,
		uow:                     uow,
		sagaRepository:          sagaRepository,
		sagaDeadline:            sagaDeadline,
	}
}
