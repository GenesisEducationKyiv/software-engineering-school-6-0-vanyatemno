package e2e

import (
	"testing"

	"se-school/tests/e2e/helpers"

	"github.com/playwright-community/playwright-go"
)

// TestList_ShowsConfirmedSubscription: after subscribe+confirm, the
// Manage Subscriptions page renders the repo for that email.
func TestList_ShowsConfirmedSubscription(t *testing.T) {
	s := helpers.NewSuite(t)
	email := "list-ok@e2e.local"

	s.SubscribeViaUI(email, "facebook/react")
	s.WaitForSubscribeSuccess()
	s.ConfirmViaMailLink(email)

	s.GoTo("/subscriptions")
	if err := s.Page.Locator(`input[type="email"]`).Fill(email); err != nil {
		t.Fatalf("fill email: %v", err)
	}
	if err := s.Page.Locator(`button[type="submit"]`).Click(); err != nil {
		t.Fatalf("click search: %v", err)
	}

	loc := s.Page.GetByText("facebook/react")
	if err := loc.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(10000),
	}); err != nil {
		t.Fatalf("expected repo not listed: %v", err)
	}

	header := s.Page.GetByText("Found 1 subscription")
	if err := header.WaitFor(); err != nil {
		t.Fatalf("expected 'Found 1 subscription' header: %v", err)
	}
}

// TestList_EmptyForUnknownEmail: searching with an email that never
// subscribed shows the empty-state.
func TestList_EmptyForUnknownEmail(t *testing.T) {
	s := helpers.NewSuite(t)

	s.GoTo("/subscriptions")
	if err := s.Page.Locator(`input[type="email"]`).Fill("nobody@e2e.local"); err != nil {
		t.Fatalf("fill email: %v", err)
	}
	if err := s.Page.Locator(`button[type="submit"]`).Click(); err != nil {
		t.Fatalf("click search: %v", err)
	}

	loc := s.Page.GetByText("No active subscriptions found")
	if err := loc.WaitFor(); err != nil {
		t.Fatalf("empty-state not visible: %v", err)
	}
}
