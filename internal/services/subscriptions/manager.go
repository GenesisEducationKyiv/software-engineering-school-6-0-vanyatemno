package subscriptions

import "se-school/internal/models"

func (s *Service) Create(subscription *models.Subscription) (*models.Subscription, error) {
	return subscription, s.db.Create(subscription).Error
}

func (s *Service) UpdateLastSeenTag(id uint, tag string) error {
	subscription, err := s.GetByID(id)
	if err != nil {
		return err
	}
	subscription.LastSeenTag = tag

	return s.db.Save(subscription).Error
}

func (s *Service) Delete(id uint) error {
	return s.db.Where("id = ?", id).Delete(&models.Subscription{}).Error
}
