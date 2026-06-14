package codes

import "se-school/internal/models"

type FactoryMock struct {
	NewErr    error
	NewResult *models.Code

	NewCalls []models.CodeType
}

func NewFactoryMock() *FactoryMock {
	return &FactoryMock{}
}

func (m *FactoryMock) New(codeType models.CodeType) (*models.Code, error) {
	m.NewCalls = append(m.NewCalls, codeType)
	if m.NewErr != nil {
		return nil, m.NewErr
	}
	if m.NewResult != nil {
		return m.NewResult, nil
	}
	return &models.Code{Type: codeType, Code: "mock-code"}, nil
}
