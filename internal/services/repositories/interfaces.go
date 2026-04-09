package repositories

import "context"

type RepositoriesManager interface {
	UpdateRepositoryVersion(ctx context.Context, id uint) error
}
