package models

import (
	"time"

	"gorm.io/gorm"
)

type CodeType = string

const (
	CodeTypeConfirm     CodeType = "confirmation"
	CodeTypeUnsubscribe CodeType = "unsubscribe"
)

type Code struct {
	gorm.Model

	Code      string    `gorm:"text;not null;unique" json:"code"`
	Type      CodeType  `gorm:"text;not null;index" json:"type"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
}

func (c *Code) Migrate(db *gorm.DB) error {
	return db.AutoMigrate(c)
}
