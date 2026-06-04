package models

import "time"

type CodeType = string

const (
	CodeTypeConfirm     CodeType = "confirmation"
	CodeTypeUnsubscribe CodeType = "unsubscribe"
)

type Code struct {
	ID        uint       `json:"ID"`
	CreatedAt time.Time  `json:"CreatedAt"`
	UpdatedAt time.Time  `json:"UpdatedAt"`
	DeletedAt *time.Time `json:"DeletedAt,omitempty"`

	Code      string    `json:"code"`
	Type      CodeType  `json:"type"`
	ExpiresAt time.Time `json:"expires_at"`
}
