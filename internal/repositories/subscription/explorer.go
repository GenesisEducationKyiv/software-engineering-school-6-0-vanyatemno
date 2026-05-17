package subscription

import (
	"errors"
	"se-school/internal/models"
	"se-school/internal/repositories"

	"gorm.io/gorm"
)

func (r *Repository) GetByID(id uint) (*models.Subscription, error) {
	var subscription models.Subscription
	err := r.db.Where("id = ?", id).First(&subscription).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	return &subscription, nil
}

func (r *Repository) GetUnupdated(repositoryID uint, currentTag string) ([]*models.Subscription, error) {
	var subscriptions []*models.Subscription
	err := r.db.
		Where("repository_id = ? AND last_seen_tag != ?", repositoryID, currentTag).
		Find(&subscriptions).
		Error
	if err != nil {
		return nil, err
	}

	return subscriptions, nil
}

func (r *Repository) GetByEmail(email string) ([]*models.Subscription, error) {
	var subscriptions []*models.Subscription
	err := r.db.
		Preload("Repository").
		Where("email = ?", email).
		Find(&subscriptions).
		Error
	if err != nil {
		return nil, err
	}

	return subscriptions, nil
}

func (r *Repository) GetByCode(codeID uint, codeType models.CodeType) (*models.Subscription, error) {
	var subscription models.Subscription
	query := r.db
	switch codeType {
	case models.CodeTypeUnsubscribe:
		query = query.Where("unsubscribe_code_id = ?", codeID)
	case models.CodeTypeConfirm:
		query = query.Where("subscribe_code_id = ?", codeID)
	default:
		return nil, errors.New("invalid codeType")
	}
	err := query.First(&subscription).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	return &subscription, nil
}
