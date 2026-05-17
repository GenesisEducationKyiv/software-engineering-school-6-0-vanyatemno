package subscription

import "se-school/internal/models"

type SubscriptionsRepository interface {
	GetByID(uint) (*models.Subscription, error)
	GetUnupdated(repositoryID uint, currentTag string) ([]*models.Subscription, error) // todo: make up better name - probably something like "unnotified" or smth
	GetByCode(codeID uint, codeType models.CodeType) (*models.Subscription, error)
	GetByEmail(string) ([]*models.Subscription, error)
	Create(subscription *models.Subscription) error
	UpdateLastSeenTag(id uint, tag string) error
	Save(*models.Subscription) error
	Delete(subscription *models.Subscription) error
}
