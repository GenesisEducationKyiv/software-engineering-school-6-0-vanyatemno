package outbox

import (
	"context"

	"se-school/internal/models"
)

// Repo is the outbox persistence contract used by the relay (and, via WithTx,
// the orchestrator's atomic create).
type Repo interface {
	Create(ctx context.Context, m *models.OutboxMessage) error
	GetUnpublished(ctx context.Context, limit int) ([]*models.OutboxMessage, error)
	MarkPublished(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id, cause string) error
}
