package codes

import (
	"strings"
	"testing"
	"time"

	"se-school/internal/models"

	"github.com/google/uuid"
)

func TestFactoryNewConfirmation(t *testing.T) {
	f := NewFactory()
	before := time.Now()

	code, err := f.New(models.CodeTypeConfirm)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if code.Type != models.CodeTypeConfirm {
		t.Fatalf("expected type %q, got %q", models.CodeTypeConfirm, code.Type)
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

func TestFactoryNewUnsubscribe(t *testing.T) {
	f := NewFactory()
	before := time.Now()

	code, err := f.New(models.CodeTypeUnsubscribe)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if code.Type != models.CodeTypeUnsubscribe {
		t.Fatalf("expected type %q, got %q", models.CodeTypeUnsubscribe, code.Type)
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

func TestFactoryNewUnknownType(t *testing.T) {
	f := NewFactory()

	code, err := f.New("unknown")
	if err == nil {
		t.Fatal("expected error for unknown code type")
	}
	if !strings.Contains(err.Error(), "unknown code type: unknown") {
		t.Fatalf("expected unknown code type error, got %v", err)
	}
	if code != nil {
		t.Fatalf("expected nil code, got %+v", code)
	}
}
