package outbox

import (
	"context"

	"se-school/internal/models"
)

// RepositoryMock is a hand-written mock of the outbox repository for unit tests.
type RepositoryMock struct {
	Created  []*models.OutboxMessage
	CreateFn func(ctx context.Context, m *models.OutboxMessage) error

	GetUnpublishedResult []*models.OutboxMessage
	GetUnpublishedErr    error

	Published        []string
	MarkPublishedErr error

	FailedCalls   []string
	MarkFailedErr error
}

func NewRepositoryMock() *RepositoryMock { return &RepositoryMock{} }

func (m *RepositoryMock) Create(ctx context.Context, msg *models.OutboxMessage) error {
	m.Created = append(m.Created, msg)
	if m.CreateFn != nil {
		return m.CreateFn(ctx, msg)
	}
	return nil
}

func (m *RepositoryMock) GetUnpublished(_ context.Context, _ int) ([]*models.OutboxMessage, error) {
	return m.GetUnpublishedResult, m.GetUnpublishedErr
}

func (m *RepositoryMock) MarkPublished(_ context.Context, id string) error {
	m.Published = append(m.Published, id)
	return m.MarkPublishedErr
}

func (m *RepositoryMock) MarkFailed(_ context.Context, id, _ string) error {
	m.FailedCalls = append(m.FailedCalls, id)
	return m.MarkFailedErr
}
