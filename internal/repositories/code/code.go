package code

import (
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
	if err != nil {
		return nil, err
	}

	return &code, nil
}

func (r *Repository) Create(
	codeType models.CodeType,
) (*models.Code, error) {
	code := models.Code{
		Code: "",
		Type: codeType,
	}
	err := r.setupCode(&code)
	if err != nil {
		return nil, err
	}

	err = r.db.Create(&code).Error
	if err != nil {
		return nil, err
	}

	return &code, nil
}

func (r *Repository) Delete(id uint) error {
	return r.db.Delete(&models.Code{}, id).Error
}
