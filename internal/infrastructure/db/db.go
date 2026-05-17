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

	err = migrate(
		database,
		&models.Subscription{},
		&models.Repository{},
		&models.Code{},
	)
	if err != nil {
		return nil, err
	}

	return database, nil
}

func migrate(db *gorm.DB, models ...models.MigratableModel) error {
	for _, model := range models {
		err := model.Migrate(db)
		if err != nil {
			return err
		}
	}

	return nil
}
