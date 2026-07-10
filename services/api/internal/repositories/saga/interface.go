package saga

import (
	"context"
	"time"

	"se-school/internal/models"
)

// Repo is the saga persistence contract (the orchestrator declares its own
// matching interface at the consumer side; this exists for parity with the
// other repositories and to document the surface).
type Repo interface {
	Create(ctx context.Context, s *models.SagaInstance) error
	GetByID(ctx context.Context, id string) (*models.SagaInstance, error)
	UpdateState(ctx context.Context, id string, state models.SagaState, lastErr string) error
	IncrementAttempts(ctx context.Context, id string) (int, error)
	GetStuck(ctx context.Context, state models.SagaState, before time.Time, limit int) ([]*models.SagaInstance, error)
}
