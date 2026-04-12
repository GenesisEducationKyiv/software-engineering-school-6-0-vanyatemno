package repository

import "se-school/internal/models"

func (r *Repository) GetByID(id uint) (*models.Repository, error) {
	var repository models.Repository
	err := r.db.Where("id = ?", id).First(&repository).Error
	if err != nil {
		return nil, err
	}

	return &repository, nil
}
