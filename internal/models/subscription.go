package models

import (
	"time"

	"gorm.io/gorm"
)

type Subscription struct {
	gorm.Model

	RepositoryID uint `gorm:"foreignKey:RepositoryID" json:"repository_id"`

	Email       string    `gorm:"type:text;not null;index" json:"email"`
	IsConfirmed bool      `gorm:"type:boolean;not null" json:"confirmed"`
	LastSeenTag time.Time `gorm:"type:timestamptz;not null" json:"last_seen_tag"`

	Repository *Repository `gorm:"foreignKey:RepositoryID" json:"repository,omitempty"`
}
