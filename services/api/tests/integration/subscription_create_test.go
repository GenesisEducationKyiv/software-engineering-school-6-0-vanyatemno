package integration

import (
	"net/http"
	"testing"

	"ghnotify/contract"

	"se-school/internal/integrations/github"
	"se-school/internal/models"
	"se-school/internal/models/dto"
	"se-school/tests/integration/helpers"
)

func TestCreate_NewRepo_PersistsAndEnqueuesCommand(t *testing.T) {
	s := helpers.NewSuite(t)

	s.GH.Get("/repos/:owner/:repo/releases/latest", func(req helpers.Request) helpers.Response {
		if req.Params["owner"] != "octocat" || req.Params["repo"] != "hello-world" {
			t.Errorf("unexpected repo: %v", req.Params)
		}
		return helpers.JSON(http.StatusOK, map[string]any{"tag_name": "v1.0.0"})
	})

	sagaID, err := s.Svc.Create(s.Ctx, &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "octocat/hello-world",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sagaID == "" {
		t.Fatal("expected a saga id")
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
	// T1 atomically wrote the saga instance and the outbox command.
	if got := s.CountSagas(t); got != 1 {
		t.Fatalf("expected 1 saga, got %d", got)
	}
	if got := s.CountOutbox(t); got != 1 {
		t.Fatalf("expected 1 outbox command, got %d", got)
	}
	if state := s.SagaState(t, sagaID); state != models.SagaStateAwaitingNotification {
		t.Fatalf("expected saga AWAITING_NOTIFICATION, got %q", state)
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

	cacheKey := github.CacheKey("octocat", "hello-world")
	if got, err := s.Redis.Get(s.Ctx, cacheKey).Result(); err != nil {
		t.Fatalf("expected redis cache key %s to be set: %v", cacheKey, err)
	} else if got != "v1.0.0" {
		t.Fatalf("expected cached version v1.0.0, got %q", got)
	}
}

func TestCreate_ExistingRepository_NoGithubCall(t *testing.T) {
	s := helpers.NewSuite(t)

	s.SeedRepository(t, "octocat", "hello-world", "v2.3.4")
	s.GH.FailOnAnyRequest("repository already exists, no github traffic expected")

	if _, err := s.Svc.Create(s.Ctx, &dto.CreateSubscriptionRequest{
		Email: "existing@example.com",
		Repo:  "octocat/hello-world",
	}); err != nil {
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

	if _, err := s.Svc.Create(s.Ctx, &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "missing/repo",
	}); err == nil {
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
	if got := s.CountSagas(t); got != 0 {
		t.Fatalf("expected 0 sagas on failure, got %d", got)
	}
	if got := s.CountOutbox(t); got != 0 {
		t.Fatalf("expected 0 outbox commands on failure, got %d", got)
	}
}

func TestCreate_DuplicateEmailRepo_SecondCallFailsAtomically(t *testing.T) {
	s := helpers.NewSuite(t)

	s.GH.Get("/repos/:owner/:repo/releases/latest", func(req helpers.Request) helpers.Response {
		return helpers.JSON(http.StatusOK, map[string]any{"tag_name": "v1.0.0"})
	})

	req := &dto.CreateSubscriptionRequest{Email: "dup@example.com", Repo: "octocat/hello-world"}
	if _, err := s.Svc.Create(s.Ctx, req); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	if _, err := s.Svc.Create(s.Ctx, req); err == nil {
		t.Fatal("expected error on duplicate subscription, got nil")
	}

	// The second call's transaction rolled back entirely: no orphan codes,
	// saga or outbox rows from the failed attempt.
	if got := s.CountSubscriptions(t); got != 1 {
		t.Fatalf("expected 1 subscription after duplicate, got %d", got)
	}
	if got := s.CountCodes(t); got != 2 {
		t.Fatalf("expected 2 codes after duplicate (no orphans), got %d", got)
	}
	if got := s.CountSagas(t); got != 1 {
		t.Fatalf("expected 1 saga after duplicate, got %d", got)
	}
	if got := s.CountOutbox(t); got != 1 {
		t.Fatalf("expected 1 outbox command after duplicate, got %d", got)
	}
}

func TestCreate_DispatchedReply_CompletesSaga(t *testing.T) {
	s := helpers.NewSuite(t)

	s.GH.Get("/repos/:owner/:repo/releases/latest", func(req helpers.Request) helpers.Response {
		return helpers.JSON(http.StatusOK, map[string]any{"tag_name": "v1.0.0"})
	})

	sagaID, err := s.Svc.Create(s.Ctx, &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "octocat/hello-world",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// The notifier reports the confirmation email was dispatched.
	if err := s.Svc.HandleReply(s.Ctx, contract.Reply{
		SagaID: sagaID,
		Status: contract.ReplyDispatched,
	}); err != nil {
		t.Fatalf("HandleReply: %v", err)
	}

	if state := s.SagaState(t, sagaID); state != models.SagaStateCompleted {
		t.Fatalf("expected saga COMPLETED, got %q", state)
	}
	// The subscription persists (unconfirmed), awaiting the user's confirm click.
	if got := s.CountSubscriptions(t); got != 1 {
		t.Fatalf("expected subscription to persist, got %d", got)
	}
	if got := s.CountCodes(t); got != 2 {
		t.Fatalf("expected 2 codes to persist, got %d", got)
	}
}

func TestCreate_FailedReply_CompensatesSubscriptionAndCodes(t *testing.T) {
	s := helpers.NewSuite(t)

	s.GH.Get("/repos/:owner/:repo/releases/latest", func(req helpers.Request) helpers.Response {
		return helpers.JSON(http.StatusOK, map[string]any{"tag_name": "v1.0.0"})
	})

	sagaID, err := s.Svc.Create(s.Ctx, &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "octocat/hello-world",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := s.CountSubscriptions(t); got != 1 {
		t.Fatalf("expected 1 subscription before compensation, got %d", got)
	}

	// The notifier reports it could not dispatch the confirmation email.
	if err := s.Svc.HandleReply(s.Ctx, contract.Reply{
		SagaID: sagaID,
		Status: contract.ReplyFailed,
		Reason: "smtp down",
	}); err != nil {
		t.Fatalf("HandleReply: %v", err)
	}

	// Compensation soft-deleted the subscription and both codes.
	if got := s.CountSubscriptions(t); got != 0 {
		t.Fatalf("expected subscription compensated away, got %d live rows", got)
	}
	if got := s.CountCodes(t); got != 0 {
		t.Fatalf("expected both codes compensated away, got %d live rows", got)
	}
	// The repository row is intentionally NOT compensated (shared upstream state).
	if got := s.CountRepositories(t); got != 1 {
		t.Fatalf("expected repository row to remain, got %d", got)
	}
	if state := s.SagaState(t, sagaID); state != models.SagaStateCompensated {
		t.Fatalf("expected saga COMPENSATED, got %q", state)
	}
}

func TestCreate_InvalidRepoFormat_NoWrites(t *testing.T) {
	s := helpers.NewSuite(t)

	s.GH.FailOnAnyRequest("invalid repo format must fail before any github traffic")

	if _, err := s.Svc.Create(s.Ctx, &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "not-a-valid-repo",
	}); err == nil {
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

	cacheKey := github.CacheKey("octocat", "hello-world")
	if err := s.Redis.Set(s.Ctx, cacheKey, "v5.5.5", 0).Err(); err != nil {
		t.Fatalf("seed redis cache: %v", err)
	}

	s.GH.FailOnAnyRequest("redis cache hit must short-circuit github HTTP call")

	if _, err := s.Svc.Create(s.Ctx, &dto.CreateSubscriptionRequest{
		Email: "cached@example.com",
		Repo:  "octocat/hello-world",
	}); err != nil {
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
