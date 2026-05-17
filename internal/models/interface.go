package models

import "gorm.io/gorm"

type MigratableModel interface {
	Migrate(db *gorm.DB) error
}
