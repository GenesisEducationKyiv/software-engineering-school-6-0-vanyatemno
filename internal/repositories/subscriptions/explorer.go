package subscription

import "se-school/internal/models"

func (r *Repository) GetByID(id uint) (*models.Subscription, error) {
	var subscription models.Subscription
	err := r.db.Where("id = ?", id).First(&subscription).Error
	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func (r *Repository) GetByRepositoryID(repositoryID uint) ([]*models.Subscription, error) {
	var subscriptions []*models.Subscription
	err := r.db.Where("repository_id = ?", repositoryID).Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}

	return subscriptions, nil
}
