package outbox

import (
	"context"

	"se-school/internal/models"
)

type Repo interface {
	Create(ctx context.Context, m *models.OutboxMessage) error
	GetUnpublished(ctx context.Context, limit int) ([]*models.OutboxMessage, error)
	MarkPublished(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id, cause string) error
}
