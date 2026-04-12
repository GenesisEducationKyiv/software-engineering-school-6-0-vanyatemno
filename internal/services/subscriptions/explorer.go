package subscriptions

import "se-school/internal/models"

func (s *Service) GetByID(id uint) (*models.Subscription, error) {
	var subscription models.Subscription
	err := s.db.Where("id = ?", id).First(&subscription).Error
	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func (s *Service) GetByRepositoryID(repositoryID uint) ([]*models.Subscription, error) {
	var subscriptions []*models.Subscription
	err := s.db.Where("repository_id = ?", repositoryID).Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}

	return subscriptions, nil
}
