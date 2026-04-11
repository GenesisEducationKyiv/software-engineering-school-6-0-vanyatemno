package models

import "gorm.io/gorm"

type Repository struct {
	gorm.Model

	Owner   string `gorm:"index:idx_repository"`
	Name    string `gorm:"index:idx_repository"`
	Path    string `gorm:"type:text;not null" json:"name"`
	Version string `gorm:"type:text;not null" json:"version"`
}
