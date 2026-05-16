package code

import "se-school/internal/models"

type CodesRepositoryMock struct {
	GetResult *models.Code
	GetErr    error
	CreateErr error
	DeleteErr error

	CreateCalls []*models.Code
	DeleteCalls []uint
}

func NewCodesRepositoryMock() *CodesRepositoryMock {
	return &CodesRepositoryMock{}
}

func (m *CodesRepositoryMock) Get(_ string) (*models.Code, error) {
	return m.GetResult, m.GetErr
}

func (m *CodesRepositoryMock) Create(code *models.Code) error {
	m.CreateCalls = append(m.CreateCalls, code)
	return m.CreateErr
}

func (m *CodesRepositoryMock) Delete(id uint) error {
	m.DeleteCalls = append(m.DeleteCalls, id)
	return m.DeleteErr
}
