package codes

import (
	"se-school/internal/models"
	"se-school/internal/utils"
	"time"

	"github.com/google/uuid"
)

type codeTypePolicy struct {
	TTL      time.Duration
	Generate func() (string, error)
}

var policies = map[models.CodeType]codeTypePolicy{
	models.CodeTypeConfirm: {
		TTL:      30 * time.Minute,
		Generate: func() (string, error) { return utils.GenerateCode(confirmationCodeLength) },
	},
	models.CodeTypeUnsubscribe: {
		TTL:      10 * 365 * 24 * time.Hour,
		Generate: func() (string, error) { return uuid.New().String(), nil },
	},
}
