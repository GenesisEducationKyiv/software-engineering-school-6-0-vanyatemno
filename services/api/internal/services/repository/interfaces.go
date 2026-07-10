package repository

import (
	"context"

	"ghnotify/contract"

	"se-school/internal/models"
)

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

// NotificationsService delivers one repository-update alert to one recipient
// synchronously. A nil return means the notifier confirmed delivery (so the
// caller may advance the subscription's last_seen_tag); a non-nil return means
// delivery failed and the tag must be left unchanged for a retry next run.
type NotificationsService interface {
	Notify(ctx context.Context, recipient string, template contract.TemplateName, data any) error
}

type GithubIntegration interface {
	GetRepositoryVersion(ctx context.Context, owner, repositoryName string) (string, error)
}
