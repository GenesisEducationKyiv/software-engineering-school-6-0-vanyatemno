package models

import (
	"gorm.io/gorm"
)

type Subscription struct {
	gorm.Model

	RepositoryID      uint `gorm:"not null" json:"repository_id"`
	SubscribeCodeID   uint `gorm:"not null" json:"-"`
	UnsubscribeCodeID uint `gorm:"not null" json:"-"`

	Email       string `gorm:"type:text;not null" json:"email"`
	IsConfirmed bool   `gorm:"type:boolean;not null" json:"confirmed"`
	LastSeenTag string `gorm:"type:text;not null" json:"last_seen_tag"`

	UnsubscribeCode *Code       `gorm:"foreignKey:UnsubscribeCodeID" json:"-"`
	SubscribeCode   *Code       `gorm:"foreignKey:SubscribeCodeID" json:"-"`
	Repository      *Repository `gorm:"foreignKey:RepositoryID" json:"repository,omitempty"`
}

func (Subscription) Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Subscription{}); err != nil {
		return err
	}

	return db.Exec(`
        CREATE UNIQUE INDEX IF NOT EXISTS unique_idx_email_repository_id
        ON subscriptions (email, repository_id)
        WHERE deleted_at IS NULL
    `).Error
}

func (s *Subscription) AfterDelete(tx *gorm.DB) error {
	if s.SubscribeCodeID != 0 {
		if err := tx.Delete(&Code{}, s.SubscribeCodeID).Error; err != nil {
			return err
		}
	}
	if s.UnsubscribeCodeID != 0 {
		if err := tx.Delete(&Code{}, s.UnsubscribeCodeID).Error; err != nil {
			return err
		}
	}
	return nil
}
