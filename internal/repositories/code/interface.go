package code

import "se-school/internal/models"

type CodesRepository interface {
	Get(code string) (*models.Code, error)
	Create(subscriptionID uint, codeType models.CodeType) error
	Delete(id uint) error
}
