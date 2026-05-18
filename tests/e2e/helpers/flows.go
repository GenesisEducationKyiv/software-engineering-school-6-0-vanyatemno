package helpers

import (
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

// SubscribeViaUI fills the Home page form and submits. Returns once the
// success or error toast appears. Caller asserts which one.
func (s *Suite) SubscribeViaUI(email, repo string) {
	s.T.Helper()
	s.GoTo("/")
	mustFill(s.T, s.Page, "#email", email)
	mustFill(s.T, s.Page, "#repo", repo)
	if err := s.Page.Locator(`button[type="submit"]`).Click(); err != nil {
		s.T.Fatalf("click subscribe: %v", err)
	}
}

// WaitForSubscribeSuccess waits for the green success toast on the Home
// page to appear and fails the test if it doesn't.
func (s *Suite) WaitForSubscribeSuccess() {
	s.T.Helper()
	loc := s.Page.GetByText("Subscription successful")
	if err := loc.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(10000),
	}); err != nil {
		s.T.Fatalf("subscribe success toast not visible: %v", err)
	}
}

// ConfirmViaMailLink polls Mailpit for the confirmation email sent to
// `email`, extracts the token, navigates to the confirm page, and waits
// for the success heading.
func (s *Suite) ConfirmViaMailLink(email string) {
	s.T.Helper()
	html := s.Mailpit.WaitForMessageTo(s.T, email, 30*time.Second)
	token, err := ExtractConfirmToken(html)
	if err != nil {
		s.T.Fatalf("extract token: %v", err)
	}
	s.GoTo("/confirm/" + token)
	loc := s.Page.GetByText("Subscription Confirmed!")
	if err := loc.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(10000),
	}); err != nil {
		s.T.Fatalf("confirm success heading not visible: %v", err)
	}
}

func mustFill(t *testing.T, p playwright.Page, selector, value string) {
	t.Helper()
	if err := p.Locator(selector).Fill(value); err != nil {
		t.Fatalf("fill %s: %v", selector, err)
	}
}
