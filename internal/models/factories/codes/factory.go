package codes

import (
	"fmt"
	"se-school/internal/models"
	"time"
)

const confirmationCodeLength = 6

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) New(codeType models.CodeType) (*models.Code, error) {
	policy, ok := policies[codeType]
	if !ok {
		return nil, fmt.Errorf("unknown code type: %s", codeType)
	}
	value, err := policy.Generate()
	if err != nil {
		return nil, err
	}
	return &models.Code{
		Code:      value,
		Type:      codeType,
		ExpiresAt: time.Now().Add(policy.TTL),
	}, nil
}
