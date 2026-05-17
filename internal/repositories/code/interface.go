package code

import (
	"context"
	"se-school/internal/models"
)

type CodesRepository interface {
	Get(ctx context.Context, code string) (*models.Code, error)
	Create(ctx context.Context, codeType models.CodeType) (*models.Code, error)
	Delete(ctx context.Context, id uint) error
}
