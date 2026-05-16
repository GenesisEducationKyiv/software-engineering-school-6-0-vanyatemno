package integration

import (
	"testing"

	"se-school/internal/models/dto"
	"se-school/tests/integration/helpers"
)

func TestConfirm_ValidToken_FlipsFlagAndDropsConfirmCode(t *testing.T) {
	s := helpers.NewSuite(t)

	sub := s.SeedSubscription(t, "user@example.com", "octocat", "hello-world", "v1.0.0", false)
	confirmCodeID := sub.SubscribeCode.ID
	unsubCodeID := sub.UnsubscribeCode.ID

	err := s.Svc.Confirm(&dto.ConfirmSubscriptionRequest{Token: sub.SubscribeCode.Code})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	stored := s.FindSubscriptionByEmail(t, "user@example.com")
	if !stored.IsConfirmed {
		t.Fatal("expected subscription.IsConfirmed to be true after confirm")
	}

	if s.CodeExists(t, confirmCodeID) {
		t.Fatalf("expected confirm code %d to be deleted after Confirm", confirmCodeID)
	}
	if !s.CodeExists(t, unsubCodeID) {
		t.Fatalf("expected unsubscribe code %d to still exist after Confirm", unsubCodeID)
	}
}

func TestConfirm_InvalidToken_ReturnsErrorAndLeavesSubscriptionUntouched(t *testing.T) {
	s := helpers.NewSuite(t)

	sub := s.SeedSubscription(t, "user@example.com", "octocat", "hello-world", "v1.0.0", false)

	err := s.Svc.Confirm(&dto.ConfirmSubscriptionRequest{Token: "this-token-does-not-exist"})
	if err == nil {
		t.Fatal("expected error for unknown confirm token, got nil")
	}

	stored := s.FindSubscriptionByEmail(t, "user@example.com")
	if stored.IsConfirmed {
		t.Fatal("expected subscription.IsConfirmed to remain false after invalid confirm")
	}
	if !s.CodeExists(t, sub.SubscribeCode.ID) {
		t.Fatal("expected confirm code to remain after invalid confirm")
	}
}

func TestConfirm_ReusedToken_SecondCallFails(t *testing.T) {
	s := helpers.NewSuite(t)

	sub := s.SeedSubscription(t, "user@example.com", "octocat", "hello-world", "v1.0.0", false)
	token := sub.SubscribeCode.Code

	if err := s.Svc.Confirm(&dto.ConfirmSubscriptionRequest{Token: token}); err != nil {
		t.Fatalf("first Confirm: %v", err)
	}

	err := s.Svc.Confirm(&dto.ConfirmSubscriptionRequest{Token: token})
	if err == nil {
		t.Fatal("expected error reusing already-consumed confirm token, got nil")
	}
}
