package models

import "gorm.io/gorm"

type Repository struct {
	gorm.Model

	Owner   string `gorm:"index:idx_repository_owner"`
	Name    string `gorm:"index:idx_repository_name"`
	Path    string `gorm:"type:text;not null" json:"name"`
	Version string `gorm:"type:text;not null" json:"version"`
}
