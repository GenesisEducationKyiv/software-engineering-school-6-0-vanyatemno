package e2e

import (
	"testing"

	"se-school/tests/e2e/helpers"
)

// TestUnsubscribe_HappyPath: after subscribe+confirm, navigating to
// /unsubscribe/<code> with the real unsubscribe code (fetched from the
// DB) flips the subscription to soft-deleted.
//
// We read the code from the database rather than from an email because
// the product wires the release-notification "Unsubscribe" link using
// the repo name, not the token. Triggering a release-notification
// email would require a release polling cycle, which is out of scope
// for the e2e flow tests.
func TestUnsubscribe_HappyPath(t *testing.T) {
	s := helpers.NewSuite(t)
	email := "unsub-ok@e2e.local"

	s.SubscribeViaUI(email, "golang/go")
	s.WaitForSubscribeSuccess()
	s.ConfirmViaMailLink(email)

	code := s.FetchUnsubscribeCode(email, "golang", "go")

	s.GoTo("/unsubscribe/" + code)
	loc := s.Page.GetByText("Unsubscribed")
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
