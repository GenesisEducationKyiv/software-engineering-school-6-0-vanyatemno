package helpers

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/playwright-community/playwright-go"
)

// Suite is the shared harness for an e2e test: browser context, page,
// http clients pointed at the stack services, and DB connection for
// truncation between tests. Created with NewSuite and torn down via
// t.Cleanup.
type Suite struct {
	T   *testing.T
	Ctx context.Context

	PW      *playwright.Playwright
	Browser playwright.Browser
	Context playwright.BrowserContext
	Page    playwright.Page

	FrontendURL string
	BackendURL  string
	APIKey      string

	DB      *pgx.Conn
	Mailpit *MailpitClient
}

func NewSuite(t *testing.T) *Suite {
	t.Helper()

	frontend := requireEnv(t, "E2E_FRONTEND_URL")
	backend := requireEnv(t, "E2E_BACKEND_URL")
	apiKey := requireEnv(t, "E2E_API_KEY")
	mailpitURL := requireEnv(t, "E2E_MAILPIT_URL")
	dbDSN := requireEnv(t, "E2E_DB_DSN")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	waitForHTTP(t, backend+"/swagger/index.html", 90*time.Second)
	waitForHTTP(t, frontend, 90*time.Second)
	waitForHTTP(t, mailpitURL+"/api/v1/info", 90*time.Second)

	conn := connectDB(t, ctx, dbDSN)
	truncate(t, ctx, conn)

	mail := &MailpitClient{baseURL: mailpitURL, http: &http.Client{Timeout: 5 * time.Second}}
	mail.DeleteAll(t)

	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("playwright run: %v", err)
	}
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		t.Fatalf("launch chromium: %v", err)
	}
	bctx, err := browser.NewContext()
	if err != nil {
		t.Fatalf("new context: %v", err)
	}
	page, err := bctx.NewPage()
	if err != nil {
		t.Fatalf("new page: %v", err)
	}

	s := &Suite{
		T: t, Ctx: ctx,
		PW: pw, Browser: browser, Context: bctx, Page: page,
		FrontendURL: frontend, BackendURL: backend, APIKey: apiKey,
		DB: conn, Mailpit: mail,
	}

	t.Cleanup(func() {
		_ = bctx.Close()
		_ = browser.Close()
		_ = pw.Stop()
		truncate(t, context.Background(), conn)
		_ = conn.Close(context.Background())
		mail.DeleteAll(t)
	})

	return s
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Fatalf("missing env %s", key)
	}
	return v
}

func waitForHTTP(t *testing.T, target string, timeout time.Duration) {
	t.Helper()
	if _, err := url.Parse(target); err != nil {
		t.Fatalf("bad URL %s: %v", target, err)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(target)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode < 500 {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("service not ready at %s", target)
}

func connectDB(t *testing.T, ctx context.Context, dsn string) *pgx.Conn {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		conn, err := pgx.Connect(ctx, dsn)
		if err == nil {
			return conn
		}
		if time.Now().After(deadline) {
			t.Fatalf("connect db: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func truncate(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()
	_, err := conn.Exec(ctx, `TRUNCATE TABLE subscriptions, repositories, codes RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// GoTo opens path on the frontend and waits for network idle.
func (s *Suite) GoTo(path string) {
	s.T.Helper()
	full := s.FrontendURL + path
	if _, err := s.Page.Goto(full, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		s.T.Fatalf("goto %s: %v", full, err)
	}
}

// Expect wraps playwright assertions tied to this suite's testing.T.
func (s *Suite) Expect() playwright.PlaywrightAssertions {
	return playwright.NewPlaywrightAssertions(5000)
}

// FetchUnsubscribeCode reads the unsubscribe code for the latest subscription
// to (email, owner, name) directly from the DB. The product wires the
// unsubscribe email Link using repo.Name rather than the code, so tests that
// drive the /unsubscribe/:token route fetch the real code here.
func (s *Suite) FetchUnsubscribeCode(email, owner, name string) string {
	s.T.Helper()
	row := s.DB.QueryRow(s.Ctx, `
		SELECT c.code
		FROM subscriptions s
		JOIN codes c          ON c.id = s.unsubscribe_code_id
		JOIN repositories r   ON r.id = s.repository_id
		WHERE s.email = $1 AND r.owner = $2 AND r.name = $3
		ORDER BY s.id DESC
		LIMIT 1`, email, owner, name)
	var code string
	if err := row.Scan(&code); err != nil {
		s.T.Fatalf("fetch unsubscribe code: %v", err)
	}
	return code
}

// CountConfirmed reports how many subscriptions for the email are confirmed.
func (s *Suite) CountConfirmed(email string) int {
	s.T.Helper()
	row := s.DB.QueryRow(s.Ctx,
		`SELECT count(*) FROM subscriptions WHERE email = $1 AND is_confirmed = true`, email)
	var n int
	if err := row.Scan(&n); err != nil {
		s.T.Fatalf("count confirmed: %v", err)
	}
	return n
}

// CountActiveSubs reports non-soft-deleted subscriptions for an email.
func (s *Suite) CountActiveSubs(email string) int {
	s.T.Helper()
	row := s.DB.QueryRow(s.Ctx,
		`SELECT count(*) FROM subscriptions WHERE email = $1 AND deleted_at IS NULL`, email)
	var n int
	if err := row.Scan(&n); err != nil {
		s.T.Fatalf("count subs: %v", err)
	}
	return n
}

func (s *Suite) Debugf(format string, args ...any) {
	s.T.Logf("[e2e] "+format, args...)
}
