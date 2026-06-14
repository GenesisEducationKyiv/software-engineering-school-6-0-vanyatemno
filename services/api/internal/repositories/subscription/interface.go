package subscription

import (
	"context"
	"se-school/internal/models"
)

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
