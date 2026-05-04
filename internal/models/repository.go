package models

import "gorm.io/gorm"

type Repository struct {
	gorm.Model

	Owner   string `gorm:"type:text;not null;index:idx_repository"`
	Name    string `gorm:"type:text;not null;index:idx_repository"`
	Version string `gorm:"type:text;not null" json:"version"`
}

func (r *Repository) Migrate(db *gorm.DB) error {
	return db.AutoMigrate(r)
}
