package saga

import (
	"context"
	"time"

	"se-school/internal/models"
)

// StateChange records one UpdateState call for assertions.
type StateChange struct {
	ID      string
	State   models.SagaState
	LastErr string
}

// RepositoryMock is a hand-written mock of the saga repository for unit tests.
type RepositoryMock struct {
	Created  []*models.SagaInstance
	CreateFn func(ctx context.Context, s *models.SagaInstance) error

	GetByIDResult *models.SagaInstance
	GetByIDErr    error

	UpdateStateCalls []StateChange
	UpdateStateFn    func(ctx context.Context, id string, state models.SagaState, lastErr string) error

	IncrementResult int
	IncrementErr    error

	GetStuckResult []*models.SagaInstance
	GetStuckErr    error
}

func NewRepositoryMock() *RepositoryMock { return &RepositoryMock{} }

func (m *RepositoryMock) Create(ctx context.Context, s *models.SagaInstance) error {
	m.Created = append(m.Created, s)
	if m.CreateFn != nil {
		return m.CreateFn(ctx, s)
	}
	return nil
}

func (m *RepositoryMock) GetByID(_ context.Context, _ string) (*models.SagaInstance, error) {
	return m.GetByIDResult, m.GetByIDErr
}

func (m *RepositoryMock) UpdateState(ctx context.Context, id string, state models.SagaState, lastErr string) error {
	m.UpdateStateCalls = append(m.UpdateStateCalls, StateChange{ID: id, State: state, LastErr: lastErr})
	if m.UpdateStateFn != nil {
		return m.UpdateStateFn(ctx, id, state, lastErr)
	}
	return nil
}

func (m *RepositoryMock) IncrementAttempts(_ context.Context, _ string) (int, error) {
	return m.IncrementResult, m.IncrementErr
}

func (m *RepositoryMock) GetStuck(_ context.Context, _ models.SagaState, _ time.Time, _ int) ([]*models.SagaInstance, error) {
	return m.GetStuckResult, m.GetStuckErr
}
