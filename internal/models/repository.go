package models

import "time"

type Repository struct {
	ID        uint       `json:"ID"`
	CreatedAt time.Time  `json:"CreatedAt"`
	UpdatedAt time.Time  `json:"UpdatedAt"`
	DeletedAt *time.Time `json:"DeletedAt,omitempty"`

	Owner   string `json:"Owner"`
	Name    string `json:"Name"`
	Version string `json:"version"`
}
