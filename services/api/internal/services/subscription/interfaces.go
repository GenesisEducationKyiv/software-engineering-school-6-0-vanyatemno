package subscription

import (
	"context"
	"time"

	"se-school/internal/models"
)

type GithubIntegration interface {
	GetRepositoryVersion(ctx context.Context, owner, repositoryName string) (string, error)
}

type CodesRepository interface {
	Get(ctx context.Context, code string) (*models.Code, error)
	Create(ctx context.Context, code *models.Code) error
	Delete(ctx context.Context, id uint) error
}

type CodeFactory interface {
	New(codeType models.CodeType) (*models.Code, error)
}

type RepositoriesRepository interface {
	GetByID(ctx context.Context, id uint) (*models.Repository, error)
	GetAll(ctx context.Context) ([]*models.Repository, error)
	Find(ctx context.Context, repo *models.Repository) (*models.Repository, error)
	Create(ctx context.Context, repo *models.Repository) error
	FindOrCreate(ctx context.Context, repo *models.Repository) (*models.Repository, error)
	UpdateTag(ctx context.Context, id uint, tag string) (*models.Repository, error)
	Delete(ctx context.Context, repo *models.Repository) error
}

type SubscriptionsRepository interface {
	GetByID(ctx context.Context, id uint) (*models.Subscription, error)
	GetUnupdated(ctx context.Context, repositoryID uint, currentTag string) ([]*models.Subscription, error)
	GetByCode(ctx context.Context, codeID uint, codeType models.CodeType) (*models.Subscription, error)
	GetByEmail(ctx context.Context, email string) ([]*models.Subscription, error)
	Create(ctx context.Context, subscription *models.Subscription) error
	UpdateLastSeenTag(ctx context.Context, id uint, tag string) error
	Save(ctx context.Context, subscription *models.Subscription) error
	Delete(ctx context.Context, subscription *models.Subscription) error
}

// SagaRepository persists the saga state machine that coordinates the
// distributed Subscribe→confirmation-email transaction.
type SagaRepository interface {
	Create(ctx context.Context, s *models.SagaInstance) error
	GetByID(ctx context.Context, id string) (*models.SagaInstance, error)
	UpdateState(ctx context.Context, id string, state models.SagaState, lastErr string) error
	GetStuck(ctx context.Context, state models.SagaState, before time.Time, limit int) ([]*models.SagaInstance, error)
}

// OutboxRepository persists outbox commands. Create participates in the atomic
// T1 via the UnitOfWork; the relay drains rows using the concrete repository.
type OutboxRepository interface {
	Create(ctx context.Context, m *models.OutboxMessage) error
}

// TxRepos is the set of repositories bound to a single transaction, handed to
// the UnitOfWork closure so all of T1's writes commit (or roll back) atomically.
type TxRepos struct {
	Subscriptions SubscriptionsRepository
	Codes         CodesRepository
	Sagas         SagaRepository
	Outbox        OutboxRepository
}

// UnitOfWork runs a function within one database transaction, exposing the
// transaction-bound repositories. It generalizes the single hand-rolled
// transaction in repositories/subscription manager.go::Delete so the
// orchestrator can persist the whole T1 atomically.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context, r *TxRepos) error) error
}
