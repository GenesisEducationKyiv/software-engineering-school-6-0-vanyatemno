package models

import "errors"

var (
	ErrNotFound           = errors.New("resource not found")
	ErrRepositoryNotFound = errors.New("github repository not found")
	ErrAlreadyExists      = errors.New("resource already exists")
)
