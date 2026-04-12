package code

import (
	"strings"
	"testing"
	"time"

	"se-school/internal/models"

	"github.com/google/uuid"
)

func TestServiceSetupCodeForConfirmation(t *testing.T) {
	service := &Repository{}
	code := &models.Code{Type: models.CodeTypeConfirmation}
	before := time.Now()

	err := service.setupCode(code)
	if err != nil {
		t.Fatalf("setupCode returned error: %v", err)
	}

	if len(code.Code) != confirmationCodeLength {
		t.Fatalf("expected confirmation code length %d, got %d", confirmationCodeLength, len(code.Code))
	}

	minExpiresAt := before.Add(30 * time.Minute)
	maxExpiresAt := time.Now().Add(30 * time.Minute)
	if code.ExpiresAt.Before(minExpiresAt) || code.ExpiresAt.After(maxExpiresAt) {
		t.Fatalf("expected expiration between %v and %v, got %v", minExpiresAt, maxExpiresAt, code.ExpiresAt)
	}
}

func TestServiceSetupCodeForUnsubscribe(t *testing.T) {
	service := &Repository{}
	code := &models.Code{Type: models.CodeTypeUnsubscribe}
	before := time.Now()

	err := service.setupCode(code)
	if err != nil {
		t.Fatalf("setupCode returned error: %v", err)
	}

	if _, err := uuid.Parse(code.Code); err != nil {
		t.Fatalf("expected unsubscribe code to be a valid UUID, got %q", code.Code)
	}

	minExpiresAt := before.Add(24 * time.Hour * 365 * 10)
	maxExpiresAt := time.Now().Add(24 * time.Hour * 365 * 10)
	if code.ExpiresAt.Before(minExpiresAt) || code.ExpiresAt.After(maxExpiresAt) {
		t.Fatalf("expected expiration between %v and %v, got %v", minExpiresAt, maxExpiresAt, code.ExpiresAt)
	}
}

func TestServiceSetupCodeReturnsErrorForUnknownType(t *testing.T) {
	service := &Repository{}
	code := &models.Code{Type: "unknown"}

	err := service.setupCode(code)
	if err == nil {
		t.Fatal("expected error for unknown code type")
	}

	if !strings.Contains(err.Error(), "unknown code type: unknown") {
		t.Fatalf("expected unknown code type error, got %v", err)
	}

	if code.Code != "" {
		t.Fatalf("expected empty code, got %q", code.Code)
	}

	if !code.ExpiresAt.IsZero() {
		t.Fatalf("expected zero expiration time, got %v", code.ExpiresAt)
	}
}
