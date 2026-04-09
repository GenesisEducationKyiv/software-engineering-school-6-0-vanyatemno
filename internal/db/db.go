package db

import (
	"se-school/internal/config"
	"se-school/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg *config.Database) (*gorm.DB, error) {
	database, err := gorm.Open(postgres.Open(cfg.DNS), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = database.AutoMigrate(&models.Subscription{})
	if err != nil {
		return nil, err
	}

	return database, nil
}
