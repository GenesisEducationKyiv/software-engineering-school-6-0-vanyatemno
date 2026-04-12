package subscription

import "se-school/internal/models"

func (r *Repository) Create(subscription *models.Subscription) error {
	return r.db.Create(subscription).Error
}

func (r *Repository) UpdateLastSeenTag(id uint, tag string) error {
	subscription, err := r.GetByID(id)
	if err != nil {
		return err
	}
	subscription.LastSeenTag = tag

	return r.db.Save(subscription).Error
}

func (r *Repository) Delete(id uint) error {
	return r.db.Where("id = ?", id).Delete(&models.Subscription{}).Error
}
