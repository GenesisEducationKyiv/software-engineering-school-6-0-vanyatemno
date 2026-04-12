package subscriptions

import "se-school/internal/models"

type SubscriptionService interface {
	GetByID(uint) (*models.Subscription, error)
	GetByRepositoryID(uint) ([]*models.Subscription, error)
	Create(subscription *models.Subscription) (*models.Subscription, error)
	UpdateLastSeenTag(id uint, tag string) error
	Delete(uint) error
}
