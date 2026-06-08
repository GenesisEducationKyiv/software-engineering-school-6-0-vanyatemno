package subscription

import (
	"context"
	"se-school/internal/models"
)

type SubscriptionsRepositoryMock struct {
	GetUnupdatedResult []*models.Subscription
	GetUnupdatedErr    error

	GetByIDResult *models.Subscription
	GetByIDErr    error

	GetByCodeResult *models.Subscription
	GetByCodeErr    error

	GetByEmailResult []*models.Subscription
	GetByEmailErr    error

	CreateErr         error
	UpdateLastSeenErr error
	SaveErr           error
	DeleteErr         error
}

func NewSubscriptionsRepositoryMock() *SubscriptionsRepositoryMock {
	return &SubscriptionsRepositoryMock{}
}

func (m *SubscriptionsRepositoryMock) GetByID(_ context.Context, _ uint) (*models.Subscription, error) {
	return m.GetByIDResult, m.GetByIDErr
}

func (m *SubscriptionsRepositoryMock) GetUnupdated(_ context.Context, _ uint, _ string) ([]*models.Subscription, error) {
	return m.GetUnupdatedResult, m.GetUnupdatedErr
}

func (m *SubscriptionsRepositoryMock) GetByCode(_ context.Context, _ uint, _ models.CodeType) (*models.Subscription, error) {
	return m.GetByCodeResult, m.GetByCodeErr
}

func (m *SubscriptionsRepositoryMock) GetByEmail(_ context.Context, _ string) ([]*models.Subscription, error) {
	return m.GetByEmailResult, m.GetByEmailErr
}

func (m *SubscriptionsRepositoryMock) Create(_ context.Context, _ *models.Subscription) error {
	return m.CreateErr
}

func (m *SubscriptionsRepositoryMock) UpdateLastSeenTag(_ context.Context, _ uint, _ string) error {
	return m.UpdateLastSeenErr
}

func (m *SubscriptionsRepositoryMock) Save(_ context.Context, _ *models.Subscription) error {
	return m.SaveErr
}

func (m *SubscriptionsRepositoryMock) Delete(_ context.Context, _ *models.Subscription) error {
	return m.DeleteErr
}
