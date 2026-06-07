package e2e

import (
	"testing"

	"se-school/tests/e2e/helpers"
)

// TestConfirm_HappyPath: subscribe via the UI, follow the confirmation
// link from the delivered email, and verify the subscription flips to
// confirmed in the database.
func TestConfirm_HappyPath(t *testing.T) {
	s := helpers.NewSuite(t)
	email := "confirm-ok@e2e.local"

	s.SubscribeViaUI(email, "cli/cli")
	s.WaitForSubscribeSuccess()
	s.ConfirmViaMailLink(email)

	if got := s.CountConfirmed(email); got != 1 {
		t.Fatalf("confirmed subs for %s = %d, want 1", email, got)
	}
}

// TestConfirm_BadToken: navigating to the confirm page with a token
// that doesn't exist surfaces the "Confirmation Failed" view.
func TestConfirm_BadToken(t *testing.T) {
	s := helpers.NewSuite(t)

	s.GoTo("/confirm/this-token-does-not-exist")

	loc := s.Page.GetByText("Confirmation Failed")
	if err := loc.WaitFor(); err != nil {
		t.Fatalf("error heading not visible: %v", err)
	}
}
