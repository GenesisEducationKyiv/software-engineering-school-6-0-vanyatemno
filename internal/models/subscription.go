package models

import "time"

type Subscription struct {
	ID        uint       `json:"ID"`
	CreatedAt time.Time  `json:"CreatedAt"`
	UpdatedAt time.Time  `json:"UpdatedAt"`
	DeletedAt *time.Time `json:"DeletedAt,omitempty"`

	RepositoryID      uint `json:"repository_id"`
	SubscribeCodeID   uint `json:"-"`
	UnsubscribeCodeID uint `json:"-"`

	Email       string `json:"email"`
	IsConfirmed bool   `json:"confirmed"`
	LastSeenTag string `json:"last_seen_tag"`

	UnsubscribeCode *Code       `json:"-"`
	SubscribeCode   *Code       `json:"-"`
	Repository      *Repository `json:"repository,omitempty"`
}
