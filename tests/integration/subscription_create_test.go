package integration

import (
	"net/http"
	"testing"

	"se-school/internal/integrations/github"
	"se-school/internal/models"
	"se-school/internal/models/dto"
	"se-school/internal/notifications/templates"
	"se-school/tests/integration/helpers"
)

func TestCreate_NewRepo_PersistsAndSendsConfirmation(t *testing.T) {
	s := helpers.NewSuite(t)

	s.GH.Get("/repos/:owner/:repo/releases/latest", func(req helpers.Request) helpers.Response {
		if req.Params["owner"] != "octocat" || req.Params["repo"] != "hello-world" {
			t.Errorf("unexpected repo: %v", req.Params)
		}
		return helpers.JSON(http.StatusOK, map[string]any{"tag_name": "v1.0.0"})
	})

	err := s.Svc.Create(s.Ctx, &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "octocat/hello-world",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if got := s.CountRepositories(t); got != 1 {
		t.Fatalf("expected 1 repository, got %d", got)
	}
	if got := s.CountSubscriptions(t); got != 1 {
		t.Fatalf("expected 1 subscription, got %d", got)
	}
	if got := s.CountCodes(t); got != 2 {
		t.Fatalf("expected 2 codes (confirm + unsubscribe), got %d", got)
	}

	sub := s.FindSubscriptionByEmail(t, "user@example.com")
	if sub.IsConfirmed {
		t.Fatal("expected new subscription to be unconfirmed")
	}
	if sub.Repository.Version != "v1.0.0" {
		t.Fatalf("expected version v1.0.0, got %q", sub.Repository.Version)
	}
	if sub.LastSeenTag != "v1.0.0" {
		t.Fatalf("expected LastSeenTag v1.0.0, got %q", sub.LastSeenTag)
	}

	if len(s.Notifier.SendEmailCalls) != 1 {
		t.Fatalf("expected 1 SendEmail call, got %d", len(s.Notifier.SendEmailCalls))
	}
	call := s.Notifier.SendEmailCalls[0]
	if call.Template != templates.Confirmation {
		t.Fatalf("expected template %q, got %q", templates.Confirmation, call.Template)
	}
	if len(call.Receivers) != 1 || call.Receivers[0] != "user@example.com" {
		t.Fatalf("expected receiver user@example.com, got %v", call.Receivers)
	}

	cacheKey := github.CacheKey("octocat", "hello-world")
	if got, err := s.Redis.Get(s.Ctx, cacheKey).Result(); err != nil {
		t.Fatalf("expected redis cache key %s to be set: %v", cacheKey, err)
	} else if got != "v1.0.0" {
		t.Fatalf("expected cached version v1.0.0, got %q", got)
	}
}

func TestCreate_ExistingRepository_NoGithubCall(t *testing.T) {
	s := helpers.NewSuite(t)

	if err := s.DB.Create(&models.Repository{
		Owner:   "octocat",
		Name:    "hello-world",
		Version: "v2.3.4",
	}).Error; err != nil {
		t.Fatalf("seed repository: %v", err)
	}

	s.GH.FailOnAnyRequest("repository already exists, no github traffic expected")

	err := s.Svc.Create(s.Ctx, &dto.CreateSubscriptionRequest{
		Email: "existing@example.com",
		Repo:  "octocat/hello-world",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if got := s.CountRepositories(t); got != 1 {
		t.Fatalf("expected still 1 repository, got %d", got)
	}
	sub := s.FindSubscriptionByEmail(t, "existing@example.com")
	if sub.LastSeenTag != "v2.3.4" {
		t.Fatalf("expected LastSeenTag v2.3.4, got %q", sub.LastSeenTag)
	}
	if s.GH.CallCount() != 0 {
		t.Fatalf("expected 0 msw calls, got %d", s.GH.CallCount())
	}
}

func TestCreate_GithubReturns404_NoPersistence(t *testing.T) {
	s := helpers.NewSuite(t)

	s.GH.Get("/repos/:owner/:repo/releases/latest", func(req helpers.Request) helpers.Response {
		return helpers.JSON(http.StatusNotFound, map[string]string{"message": "Not Found"})
	})

	err := s.Svc.Create(s.Ctx, &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "missing/repo",
	})
	if err == nil {
		t.Fatal("expected error from github 404, got nil")
	}

	if got := s.CountSubscriptions(t); got != 0 {
		t.Fatalf("expected 0 subscriptions on failure, got %d", got)
	}
	if got := s.CountRepositories(t); got != 0 {
		t.Fatalf("expected 0 repositories on failure, got %d", got)
	}
	if got := s.CountCodes(t); got != 0 {
		t.Fatalf("expected 0 codes on failure, got %d", got)
	}
	if len(s.Notifier.SendEmailCalls) != 0 {
		t.Fatalf("expected 0 emails on failure, got %d", len(s.Notifier.SendEmailCalls))
	}
}

func TestCreate_DuplicateEmailRepo_SecondCallFails(t *testing.T) {
	s := helpers.NewSuite(t)

	s.GH.Get("/repos/:owner/:repo/releases/latest", func(req helpers.Request) helpers.Response {
		return helpers.JSON(http.StatusOK, map[string]any{"tag_name": "v1.0.0"})
	})

	req := &dto.CreateSubscriptionRequest{
		Email: "dup@example.com",
		Repo:  "octocat/hello-world",
	}
	if err := s.Svc.Create(s.Ctx, req); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	err := s.Svc.Create(s.Ctx, req)
	if err == nil {
		t.Fatal("expected error on duplicate subscription, got nil")
	}

	if got := s.CountSubscriptions(t); got != 1 {
		t.Fatalf("expected 1 subscription after duplicate, got %d", got)
	}
	if len(s.Notifier.SendEmailCalls) != 1 {
		t.Fatalf("expected only 1 confirmation email after duplicate, got %d", len(s.Notifier.SendEmailCalls))
	}
}

func TestCreate_InvalidRepoFormat_NoWrites(t *testing.T) {
	s := helpers.NewSuite(t)

	s.GH.FailOnAnyRequest("invalid repo format must fail before any github traffic")

	err := s.Svc.Create(s.Ctx, &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "not-a-valid-repo",
	})
	if err == nil {
		t.Fatal("expected error for invalid repo format, got nil")
	}
	if got := s.CountSubscriptions(t); got != 0 {
		t.Fatalf("expected 0 subscriptions, got %d", got)
	}
	if got := s.CountRepositories(t); got != 0 {
		t.Fatalf("expected 0 repositories, got %d", got)
	}
}

func TestCreate_RedisCacheShortCircuitsGithub(t *testing.T) {
	s := helpers.NewSuite(t)

	// Pre-warm the cache as if a prior request had populated it. The
	// repository row does NOT exist in the DB yet, so the service will
	// route through the github integration, which must hit redis first
	// and skip the HTTP call entirely.
	cacheKey := github.CacheKey("octocat", "hello-world")
	if err := s.Redis.Set(s.Ctx, cacheKey, "v5.5.5", 0).Err(); err != nil {
		t.Fatalf("seed redis cache: %v", err)
	}

	s.GH.FailOnAnyRequest("redis cache hit must short-circuit github HTTP call")

	err := s.Svc.Create(s.Ctx, &dto.CreateSubscriptionRequest{
		Email: "cached@example.com",
		Repo:  "octocat/hello-world",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	sub := s.FindSubscriptionByEmail(t, "cached@example.com")
	if sub.Repository.Version != "v5.5.5" {
		t.Fatalf("expected cached version v5.5.5 to flow into repository row, got %q", sub.Repository.Version)
	}
	if s.GH.CallCount() != 0 {
		t.Fatalf("expected 0 msw calls (cache hit), got %d", s.GH.CallCount())
	}
}

