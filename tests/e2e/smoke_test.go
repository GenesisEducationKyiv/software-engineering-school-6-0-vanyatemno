package e2e

import (
	"strings"
	"testing"

	"se-school/tests/e2e/helpers"
)

// TestSmoke_StackUp is the cheapest sanity check: every service is
// reachable, the frontend serves the SPA shell, and the browser can
// render the home page. If this fails, every other test will too —
// running it first gives a clear "stack not up" signal.
func TestSmoke_StackUp(t *testing.T) {
	s := helpers.NewSuite(t)

	s.GoTo("/")

	title, err := s.Page.Locator("h1").First().TextContent()
	if err != nil {
		t.Fatalf("read h1: %v", err)
	}
	if !strings.Contains(title, "Never Miss a Release") {
		t.Fatalf("home h1 was %q, want it to contain 'Never Miss a Release'", title)
	}
}
