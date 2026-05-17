package code

import (
	"fmt"
	"se-school/internal/models"
	"se-school/internal/utils"
	"time"

	"github.com/google/uuid"
)

const confirmationCodeLength = 6

func (r *Repository) setupCode(code *models.Code) error {
	err := r.setCodeExpiresAt(code)
	if err != nil {
		return err
	}

	err = r.generateCode(code)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) setCodeExpiresAt(code *models.Code) error {
	switch code.Type {
	case models.CodeTypeConfirm:
		code.ExpiresAt = time.Now().Add(time.Minute * 30)
	case models.CodeTypeUnsubscribe:
		code.ExpiresAt = time.Now().Add(time.Hour * 24 * 365 * 10)
	default:
		return fmt.Errorf("unknown code type: %s", code.Type)
	}

	return nil
}

func (r *Repository) generateCode(code *models.Code) error {
	switch code.Type {
	case models.CodeTypeConfirm:
		generatedCode, err := utils.GenerateCode(confirmationCodeLength)
		if err != nil {
			return err
		}
		code.Code = generatedCode
	case models.CodeTypeUnsubscribe:
		code.Code = uuid.New().String()
	default:
		return fmt.Errorf("unknown code type: %s", code.Type)
	}

	return nil
}
