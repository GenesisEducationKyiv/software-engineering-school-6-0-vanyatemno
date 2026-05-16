package integration

import (
	"testing"

	"se-school/internal/models"
	"se-school/internal/models/dto"
	"se-school/tests/integration/helpers"
)

func TestUnsubscribe_ValidToken_SoftDeletesSubAndDropsCodes(t *testing.T) {
	s := helpers.NewSuite(t)

	sub := s.SeedSubscription(t, "user@example.com", "octocat", "hello-world", "v1.0.0", true)
	confirmCodeID := sub.SubscribeCode.ID
	unsubCodeID := sub.UnsubscribeCode.ID

	err := s.Svc.Unsubscribe(&dto.UnsubscribeRequest{Token: sub.UnsubscribeCode.Code})
	if err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}

	// Soft-delete: row should be gone for the default scope.
	var live int64
	if err := s.DB.Model(&models.Subscription{}).Where("id = ?", sub.ID).Count(&live).Error; err != nil {
		t.Fatalf("count live subscription: %v", err)
	}
	if live != 0 {
		t.Fatalf("expected subscription to be soft-deleted, found %d live rows", live)
	}

	// Both codes are deleted by the AfterDelete hook.
	if s.CodeExists(t, confirmCodeID) {
		t.Fatalf("expected confirm code %d to be deleted after unsubscribe", confirmCodeID)
	}
	if s.CodeExists(t, unsubCodeID) {
		t.Fatalf("expected unsubscribe code %d to be deleted after unsubscribe", unsubCodeID)
	}
}

func TestUnsubscribe_InvalidToken_LeavesSubscriptionInPlace(t *testing.T) {
	s := helpers.NewSuite(t)

	sub := s.SeedSubscription(t, "user@example.com", "octocat", "hello-world", "v1.0.0", true)

	err := s.Svc.Unsubscribe(&dto.UnsubscribeRequest{Token: "nonexistent-token"})
	if err == nil {
		t.Fatal("expected error for unknown unsubscribe token, got nil")
	}

	if got := s.CountSubscriptions(t); got != 1 {
		t.Fatalf("expected subscription to remain, got %d live rows", got)
	}
	if !s.CodeExists(t, sub.SubscribeCode.ID) || !s.CodeExists(t, sub.UnsubscribeCode.ID) {
		t.Fatal("expected both codes to remain after invalid unsubscribe")
	}
}

func TestUnsubscribe_ConfirmCodeRejected(t *testing.T) {
	s := helpers.NewSuite(t)

	sub := s.SeedSubscription(t, "user@example.com", "octocat", "hello-world", "v1.0.0", true)

	// Passing the confirmation token to Unsubscribe must not match: the
	// service scopes the GetByCode lookup to code type Unsubscribe.
	err := s.Svc.Unsubscribe(&dto.UnsubscribeRequest{Token: sub.SubscribeCode.Code})
	if err == nil {
		t.Fatal("expected error when passing confirm token to Unsubscribe, got nil")
	}

	if got := s.CountSubscriptions(t); got != 1 {
		t.Fatalf("expected subscription to remain, got %d live rows", got)
	}
}
