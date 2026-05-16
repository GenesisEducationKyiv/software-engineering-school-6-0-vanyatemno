package subscription

import (
	"se-school/internal/infrastructure/db"
	"se-school/internal/models"
)

func (r *Repository) Create(subscription *models.Subscription) error {
	err := r.db.Create(subscription).Error
	if db.IsDuplicateKeyError(err) {
		return models.ErrAlreadyExists
	}
	return err
}

func (r *Repository) UpdateLastSeenTag(id uint, tag string) error {
	subscription, err := r.GetByID(id)
	if err != nil {
		return err
	}
	subscription.LastSeenTag = tag

	return r.db.Save(subscription).Error
}

func (r *Repository) Save(subscription *models.Subscription) error {
	return r.db.Save(subscription).Error
}

func (r *Repository) Delete(subscription *models.Subscription) error {
	return r.db.Delete(subscription).Error
}
