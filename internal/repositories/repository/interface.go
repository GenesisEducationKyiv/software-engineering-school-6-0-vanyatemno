package repository

import "se-school/internal/models"

type RepositoriesRepository interface {
	GetByID(uint) (*models.Repository, error)
	Create(*models.Repository) error
	FindOrCreate(*models.Repository) (*models.Repository, error)
	UpdateTag(id uint, tag string) (*models.Repository, error)
	Delete(*models.Repository) error
}
