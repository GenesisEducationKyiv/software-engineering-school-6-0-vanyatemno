package code

import (
	"errors"
	"se-school/internal/models"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Get(codeString string) (*models.Code, error) {
	var code models.Code
	err := r.db.Where(&models.Code{Code: codeString}).First(&code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &code, nil
}

func (r *Repository) Create(code *models.Code) error {
	return r.db.Create(code).Error
}

func (r *Repository) Delete(id uint) error {
	return r.db.Delete(&models.Code{}, id).Error
}
