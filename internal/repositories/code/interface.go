package code

import "se-school/internal/models"

type CodesRepository interface {
	Get(code string) (*models.Code, error)
	Create(codeType models.CodeType) (*models.Code, error)
	Delete(id uint) error
}
