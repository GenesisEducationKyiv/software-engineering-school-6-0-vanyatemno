package subscription

import (
	"context"
	"se-school/internal/models"
	"se-school/internal/notifications/templates"
)

type NotificationsService interface {
	SendEmail(receivers []string, template templates.TemplateName, data any) error
}

type GithubIntegration interface {
	GetRepositoryVersion(ctx context.Context, owner, repositoryName string) (string, error)
}

type CodesRepository interface {
	Get(code string) (*models.Code, error)
	Create(codeType models.CodeType) (*models.Code, error)
	Delete(id uint) error
}

type RepositoriesRepository interface {
	GetByID(uint) (*models.Repository, error)
	GetAll() ([]*models.Repository, error)
	Find(*models.Repository) (*models.Repository, error)
	Create(*models.Repository) error
	FindOrCreate(*models.Repository) (*models.Repository, error)
	UpdateTag(id uint, tag string) (*models.Repository, error)
	Delete(*models.Repository) error
}

type SubscriptionsRepository interface {
	GetByID(uint) (*models.Subscription, error)
	GetUnupdated(repositoryID uint, currentTag string) ([]*models.Subscription, error)
	GetByCode(codeID uint, codeType models.CodeType) (*models.Subscription, error)
	GetByEmail(string) ([]*models.Subscription, error)
	Create(subscription *models.Subscription) error
	UpdateLastSeenTag(id uint, tag string) error
	Save(*models.Subscription) error
	Delete(subscription *models.Subscription) error
}
