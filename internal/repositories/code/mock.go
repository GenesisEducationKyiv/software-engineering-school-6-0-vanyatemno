package code

import "se-school/internal/models"

type CodesRepositoryMock struct {
	GetResult    *models.Code
	GetErr       error
	CreateResult *models.Code
	CreateErr    error
	DeleteErr    error

	CreateCalls []models.CodeType
	DeleteCalls []uint
}

func NewCodesRepositoryMock() *CodesRepositoryMock {
	return &CodesRepositoryMock{}
}

func (m *CodesRepositoryMock) Get(_ string) (*models.Code, error) {
	return m.GetResult, m.GetErr
}

func (m *CodesRepositoryMock) Create(codeType models.CodeType) (*models.Code, error) {
	m.CreateCalls = append(m.CreateCalls, codeType)
	if m.CreateErr != nil {
		return nil, m.CreateErr
	}
	if m.CreateResult != nil {
		return m.CreateResult, nil
	}
	return &models.Code{Type: codeType, Code: "mock-code"}, nil
}

func (m *CodesRepositoryMock) Delete(id uint) error {
	m.DeleteCalls = append(m.DeleteCalls, id)
	return m.DeleteErr
}
