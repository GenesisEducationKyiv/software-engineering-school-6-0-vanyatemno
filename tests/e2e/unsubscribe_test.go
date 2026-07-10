package e2e

import (
	"testing"

	"se-school/tests/e2e/helpers"

	"github.com/mxschmitt/playwright-go"
)

// TestUnsubscribe_HappyPath: subscribe+confirm, then unsubscribe via the code
// soft-deletes the subscription. The code is read from the DB, not an email,
// because the release-notification unsubscribe link is keyed by repo name.
func TestUnsubscribe_HappyPath(t *testing.T) {
	s := helpers.NewSuite(t)
	email := "unsub-ok@e2e.local"

	s.SubscribeViaUI(email, "cli/cli")
	s.WaitForSubscribeSuccess()
	s.ConfirmViaMailLink(email)

	code := s.FetchUnsubscribeCode(email, "cli", "cli")

	s.GoTo("/unsubscribe/" + code)
	loc := s.Page.Locator("h2", playwright.PageLocatorOptions{HasText: "Unsubscribed"})
	if err := loc.WaitFor(); err != nil {
		t.Fatalf("unsubscribe success heading not visible: %v", err)
	}

	if got := s.CountActiveSubs(email); got != 0 {
		t.Fatalf("active subs for %s after unsubscribe = %d, want 0", email, got)
	}
}

func TestUnsubscribe_BadToken(t *testing.T) {
	s := helpers.NewSuite(t)

	s.GoTo("/unsubscribe/this-token-does-not-exist")

	loc := s.Page.GetByText("Unsubscribe Failed")
	if err := loc.WaitFor(); err != nil {
		t.Fatalf("unsubscribe error heading not visible: %v", err)
	}
}
