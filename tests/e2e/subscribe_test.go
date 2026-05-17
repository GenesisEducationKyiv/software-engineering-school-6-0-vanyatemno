package e2e

import (
	"testing"
	"time"

	"se-school/tests/e2e/helpers"

	"github.com/playwright-community/playwright-go"
)

func TestSubscribe_ValidRepo_CreatesUnconfirmedSubscription(t *testing.T) {
	s := helpers.NewSuite(t)
	email := "subscribe-ok@e2e.local"

	s.SubscribeViaUI(email, "golang/go")
	s.WaitForSubscribeSuccess()

	if got := s.CountActiveSubs(email); got != 1 {
		t.Fatalf("active subs for %s = %d, want 1", email, got)
	}
	if got := s.CountConfirmed(email); got != 0 {
		t.Fatalf("confirmed subs for %s = %d, want 0 (pending)", email, got)
	}

	// Mailpit should have received the confirmation email.
	html := s.Mailpit.WaitForMessageTo(t, email, 20*time.Second)
	if _, err := helpers.ExtractConfirmToken(html); err != nil {
		t.Fatalf("no confirm token in delivered email: %v", err)
	}
}

func TestSubscribe_UnknownRepo_ShowsError(t *testing.T) {
	s := helpers.NewSuite(t)
	email := "subscribe-bad@e2e.local"

	s.SubscribeViaUI(email, "this-owner-does-not-exist-9z9z9z/nope")

	// The Home page renders a red error toast with the API error message.
	loc := s.Page.GetByText("not found", playwright.PageGetByTextOptions{Exact: playwright.Bool(false)})
	if err := loc.First().WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(10000),
	}); err != nil {
		t.Fatalf("error toast not visible: %v", err)
	}

	if got := s.CountActiveSubs(email); got != 0 {
		t.Fatalf("active subs for %s = %d, want 0 (no row should be created)", email, got)
	}
}
