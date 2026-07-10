package repository

import (
	"context"
	"errors"
	"testing"

	"ghnotify/contract"

	"se-school/internal/integrations/github"
	"se-school/internal/metrics"
	"se-school/internal/models"
	"se-school/internal/notifications"
	repoRepo "se-school/internal/repositories/repository"
	subRepo "se-school/internal/repositories/subscription"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func newTestService(
	githubVersion string,
	repos map[uint]*models.Repository,
	subs []*models.Subscription,
) (*Service, *repoRepo.RepositoriesRepositoryMock, *subRepo.SubscriptionsRepositoryMock, *notifications.NotificationsServiceMock, *github.GithubIntegrationMock) {
	repoMock := repoRepo.NewRepositoriesRepositoryMock()
	for id, r := range repos {
		repoMock.Repositories[id] = r
	}

	subMock := subRepo.NewSubscriptionsRepositoryMock()
	subMock.GetUnupdatedResult = subs

	notifMock := notifications.NewNotificationsServiceMock()
	githubMock := github.NewGithubIntegrationMock(githubVersion)

	svc := New("", repoMock, subMock, notifMock, githubMock)
	return svc, repoMock, subMock, notifMock, githubMock
}

// When the repo version is unchanged and no subscription lags behind, nothing is
// updated and nobody is notified.
func TestCheckRepoTagAndAlert_VersionUnchanged_NoUnupdatedSubs_DoesNothing(t *testing.T) {
	repo := &models.Repository{ID: 1, Owner: "owner", Name: "repo", Version: "v1.0.0"}

	svc, repoMock, subMock, notifMock, _ := newTestService("v1.0.0", map[uint]*models.Repository{1: repo}, nil)

	if err := svc.CheckRepoTagAndAlert(context.Background(), repo); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repoMock.UpdateTagCalls) != 0 {
		t.Fatalf("expected no UpdateTag calls, got %d", len(repoMock.UpdateTagCalls))
	}
	if len(notifMock.NotifyCalls) != 0 {
		t.Fatalf("expected no Notify calls, got %d", len(notifMock.NotifyCalls))
	}
	if len(subMock.UpdateLastSeenCalls) != 0 {
		t.Fatalf("expected no last_seen_tag updates, got %d", len(subMock.UpdateLastSeenCalls))
	}
}

// The key retry property: even when the repo version is already current (so no
// UpdateTag happens), a subscription still lagging behind — e.g. because its
// previous alert failed to deliver — is notified again and its tag advanced.
func TestCheckRepoTagAndAlert_VersionUnchanged_LaggingSubscriberRetried(t *testing.T) {
	repo := &models.Repository{ID: 1, Owner: "owner", Name: "repo", Version: "v2.0.0"}
	subs := []*models.Subscription{{ID: 7, Email: "alice@example.com", LastSeenTag: "v1.0.0"}}

	svc, repoMock, subMock, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, subs)

	if err := svc.CheckRepoTagAndAlert(context.Background(), repo); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repoMock.UpdateTagCalls) != 0 {
		t.Fatalf("expected no UpdateTag calls (version unchanged), got %d", len(repoMock.UpdateTagCalls))
	}
	if len(notifMock.NotifyCalls) != 1 {
		t.Fatalf("expected 1 Notify call, got %d", len(notifMock.NotifyCalls))
	}
	if notifMock.NotifyCalls[0].Recipient != "alice@example.com" {
		t.Fatalf("unexpected recipient: %s", notifMock.NotifyCalls[0].Recipient)
	}
	if len(subMock.UpdateLastSeenCalls) != 1 ||
		subMock.UpdateLastSeenCalls[0] != (subRepo.UpdateLastSeenCall{ID: 7, Tag: "v2.0.0"}) {
		t.Fatalf("expected last_seen_tag advanced to v2.0.0 for sub 7, got %+v", subMock.UpdateLastSeenCalls)
	}
}

// A new release notifies each subscriber individually (not one batched call) and
// advances each subscription's last_seen_tag on success.
func TestCheckRepoTagAndAlert_VersionChanged_NotifiesEachSubscriberAndAdvancesTag(t *testing.T) {
	repo := &models.Repository{ID: 1, Owner: "owner", Name: "repo", Version: "v1.0.0"}
	subs := []*models.Subscription{
		{ID: 1, Email: "alice@example.com"},
		{ID: 2, Email: "bob@example.com"},
	}

	svc, repoMock, subMock, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, subs)

	if err := svc.CheckRepoTagAndAlert(context.Background(), repo); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repoMock.UpdateTagCalls) != 1 ||
		repoMock.UpdateTagCalls[0].ID != 1 || repoMock.UpdateTagCalls[0].Tag != "v2.0.0" {
		t.Fatalf("expected UpdateTag(1, v2.0.0), got %+v", repoMock.UpdateTagCalls)
	}

	if len(notifMock.NotifyCalls) != 2 {
		t.Fatalf("expected 2 Notify calls (one per subscriber), got %d", len(notifMock.NotifyCalls))
	}
	for _, call := range notifMock.NotifyCalls {
		if call.Template != contract.RepositoryUpdated {
			t.Fatalf("expected template %q, got %q", contract.RepositoryUpdated, call.Template)
		}
		payload, ok := call.Data.(contract.RepositoryUpdateEmailPayload)
		if !ok {
			t.Fatalf("expected RepositoryUpdateEmailPayload, got %T", call.Data)
		}
		if payload.Owner != "owner" || payload.Name != "repo" || payload.Version != "v2.0.0" {
			t.Fatalf("unexpected payload: %+v", payload)
		}
	}

	if len(subMock.UpdateLastSeenCalls) != 2 {
		t.Fatalf("expected 2 last_seen_tag updates, got %d", len(subMock.UpdateLastSeenCalls))
	}
	for _, c := range subMock.UpdateLastSeenCalls {
		if c.Tag != "v2.0.0" {
			t.Fatalf("expected tag advanced to v2.0.0, got %q for sub %d", c.Tag, c.ID)
		}
	}
}

// The core requirement: when delivery to a subscriber fails, its last_seen_tag is
// left unchanged so it is retried next run, and the cron run itself still returns
// nil (the error is logged, not propagated).
func TestCheckRepoTagAndAlert_NotifyFails_LeavesTagUnadvanced(t *testing.T) {
	repo := &models.Repository{ID: 1, Owner: "owner", Name: "repo", Version: "v1.0.0"}
	subs := []*models.Subscription{{ID: 5, Email: "alice@example.com"}}

	svc, _, subMock, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, subs)
	notifMock.NotifyErr = errors.New("notifier unavailable")

	if err := svc.CheckRepoTagAndAlert(context.Background(), repo); err != nil {
		t.Fatalf("expected nil (error is logged, not returned), got %v", err)
	}

	if len(notifMock.NotifyCalls) != 1 {
		t.Fatalf("expected 1 Notify attempt, got %d", len(notifMock.NotifyCalls))
	}
	if len(subMock.UpdateLastSeenCalls) != 0 {
		t.Fatalf("expected last_seen_tag NOT advanced on failure, got %+v", subMock.UpdateLastSeenCalls)
	}
}

// A single unreachable subscriber does not stop the others: the reachable one is
// delivered and advanced, the failed one is left for retry.
func TestCheckRepoTagAndAlert_PartialNotifyFailure_IsolatesFailure(t *testing.T) {
	repo := &models.Repository{ID: 1, Owner: "owner", Name: "repo", Version: "v1.0.0"}
	subs := []*models.Subscription{
		{ID: 1, Email: "alice@example.com"},
		{ID: 2, Email: "bob@example.com"},
	}

	svc, _, subMock, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, subs)
	notifMock.NotifyErrFor = map[string]error{"bob@example.com": errors.New("smtp refused")}

	if err := svc.CheckRepoTagAndAlert(context.Background(), repo); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	if len(notifMock.NotifyCalls) != 2 {
		t.Fatalf("expected both subscribers attempted, got %d", len(notifMock.NotifyCalls))
	}
	if len(subMock.UpdateLastSeenCalls) != 1 ||
		subMock.UpdateLastSeenCalls[0] != (subRepo.UpdateLastSeenCall{ID: 1, Tag: "v2.0.0"}) {
		t.Fatalf("expected only alice (sub 1) advanced to v2.0.0, got %+v", subMock.UpdateLastSeenCalls)
	}
}

func TestCheckRepoTagAndAlert_GithubError_ReturnsError(t *testing.T) {
	repo := &models.Repository{ID: 1, Owner: "owner", Name: "repo", Version: "v1.0.0"}

	svc, _, _, _, githubMock := newTestService("", map[uint]*models.Repository{1: repo}, nil)
	githubMock.SetErrToReturn(errors.New("github api unavailable"))

	err := svc.CheckRepoTagAndAlert(context.Background(), repo)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "github api unavailable" {
		t.Fatalf("expected 'github api unavailable', got %q", err.Error())
	}
}

func TestCheckRepoTagAndAlert_UpdateTagError_ReturnsError(t *testing.T) {
	repo := &models.Repository{ID: 1, Owner: "owner", Name: "repo", Version: "v1.0.0"}

	svc, repoMock, _, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, nil)
	repoMock.UpdateTagErr = errors.New("db write failed")

	err := svc.CheckRepoTagAndAlert(context.Background(), repo)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db write failed" {
		t.Fatalf("expected 'db write failed', got %q", err.Error())
	}
	if len(notifMock.NotifyCalls) != 0 {
		t.Fatalf("expected no Notify calls after UpdateTag failure, got %d", len(notifMock.NotifyCalls))
	}
}

func TestCheckRepoTagAndAlert_GetUnupdatedError_ReturnsNil(t *testing.T) {
	repo := &models.Repository{ID: 1, Owner: "owner", Name: "repo", Version: "v1.0.0"}

	svc, _, subMock, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, nil)
	subMock.GetUnupdatedErr = errors.New("subscription query failed")

	err := svc.CheckRepoTagAndAlert(context.Background(), repo)
	if err != nil {
		t.Fatalf("expected nil (error is logged, not returned), got %v", err)
	}
	if len(notifMock.NotifyCalls) != 0 {
		t.Fatalf("expected no Notify calls after GetUnupdated failure, got %d", len(notifMock.NotifyCalls))
	}
}

func TestCheckRepoTagAndAlert_NoSubscribers_NoNotifications(t *testing.T) {
	repo := &models.Repository{ID: 1, Owner: "owner", Name: "repo", Version: "v1.0.0"}

	svc, repoMock, _, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, []*models.Subscription{})

	if err := svc.CheckRepoTagAndAlert(context.Background(), repo); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repoMock.UpdateTagCalls) != 1 {
		t.Fatalf("expected 1 UpdateTag call, got %d", len(repoMock.UpdateTagCalls))
	}
	if len(notifMock.NotifyCalls) != 0 {
		t.Fatalf("expected no Notify calls with an empty subscriber list, got %d", len(notifMock.NotifyCalls))
	}
}

// last_seen_tag is only advanced when the tag write itself succeeds; a store
// failure after delivery is counted but does not crash the run.
func TestCheckRepoTagAndAlert_TagWriteError_ReturnsNil(t *testing.T) {
	repo := &models.Repository{ID: 1, Owner: "owner", Name: "repo", Version: "v1.0.0"}
	subs := []*models.Subscription{{ID: 3, Email: "alice@example.com"}}

	svc, _, subMock, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, subs)
	subMock.UpdateLastSeenErr = errors.New("db write failed")

	if err := svc.CheckRepoTagAndAlert(context.Background(), repo); err != nil {
		t.Fatalf("expected nil (error is logged, not returned), got %v", err)
	}
	if len(notifMock.NotifyCalls) != 1 {
		t.Fatalf("expected 1 Notify call, got %d", len(notifMock.NotifyCalls))
	}
	// The tag write was attempted (and failed).
	if len(subMock.UpdateLastSeenCalls) != 1 {
		t.Fatalf("expected 1 UpdateLastSeenTag attempt, got %d", len(subMock.UpdateLastSeenCalls))
	}
}

func TestCheckAllReposTagAndAlert_ProcessesAllRepositories(t *testing.T) {
	repos := map[uint]*models.Repository{
		1: {ID: 1, Owner: "owner1", Name: "repo1", Version: "v1.0.0"},
		2: {ID: 2, Owner: "owner2", Name: "repo2", Version: "v1.0.0"},
	}

	svc, repoMock, _, _, _ := newTestService("v2.0.0", repos, []*models.Subscription{})

	if err := svc.CheckAllReposTagAndAlert(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repoMock.UpdateTagCalls) != 2 {
		t.Fatalf("expected 2 UpdateTag calls, got %d", len(repoMock.UpdateTagCalls))
	}
}

func TestCheckAllReposTagAndAlert_GetAllError_ReturnsError(t *testing.T) {
	svc, repoMock, _, _, _ := newTestService("v2.0.0", nil, nil)
	repoMock.GetAllErr = errors.New("db connection lost")

	err := svc.CheckAllReposTagAndAlert(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db connection lost" {
		t.Fatalf("expected 'db connection lost', got %q", err.Error())
	}
}

func TestCheckAllReposTagAndAlert_PartialFailure_ContinuesProcessing(t *testing.T) {
	repos := map[uint]*models.Repository{
		1: {ID: 1, Owner: "owner1", Name: "repo1", Version: "v1.0.0"},
		2: {ID: 2, Owner: "owner2", Name: "repo2", Version: "v1.0.0"},
	}

	svc, _, _, _, githubMock := newTestService("", repos, []*models.Subscription{})
	githubMock.SetErrToReturn(errors.New("github rate limited"))

	if err := svc.CheckAllReposTagAndAlert(context.Background()); err != nil {
		t.Fatalf("expected nil (errors are collected but not returned), got %v", err)
	}
}

func TestCheckAllReposTagAndAlert_RecordsRepoCheckSuccessMetric(t *testing.T) {
	repos := map[uint]*models.Repository{
		1: {ID: 1, Owner: "owner1", Name: "repo1", Version: "v1.0.0"},
		2: {ID: 2, Owner: "owner2", Name: "repo2", Version: "v1.0.0"},
	}

	svc, _, _, _, _ := newTestService("v2.0.0", repos, []*models.Subscription{})

	before := testutil.ToFloat64(metrics.RepoCheckTotal.WithLabelValues("success"))
	if err := svc.CheckAllReposTagAndAlert(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	after := testutil.ToFloat64(metrics.RepoCheckTotal.WithLabelValues("success"))

	if got := after - before; got != 2 {
		t.Fatalf("expected 2 success repo-check increments, got %v", got)
	}
}

func TestCheckAllReposTagAndAlert_RecordsRepoCheckErrorMetric(t *testing.T) {
	repos := map[uint]*models.Repository{
		1: {ID: 1, Owner: "owner1", Name: "repo1", Version: "v1.0.0"},
		2: {ID: 2, Owner: "owner2", Name: "repo2", Version: "v1.0.0"},
	}

	svc, _, _, _, githubMock := newTestService("", repos, []*models.Subscription{})
	githubMock.SetErrToReturn(errors.New("github rate limited"))

	before := testutil.ToFloat64(metrics.RepoCheckTotal.WithLabelValues("error"))
	if err := svc.CheckAllReposTagAndAlert(context.Background()); err != nil {
		t.Fatalf("expected nil (errors are collected but not returned), got %v", err)
	}
	after := testutil.ToFloat64(metrics.RepoCheckTotal.WithLabelValues("error"))

	if got := after - before; got != 2 {
		t.Fatalf("expected 2 error repo-check increments, got %v", got)
	}
}

// Per-subscription outcomes are recorded: a delivered subscriber increments
// "success" and a failed one increments "error".
func TestCheckRepoTagAndAlert_RecordsNotifyMetrics(t *testing.T) {
	repo := &models.Repository{ID: 1, Owner: "owner", Name: "repo", Version: "v1.0.0"}
	subs := []*models.Subscription{
		{ID: 1, Email: "alice@example.com"},
		{ID: 2, Email: "bob@example.com"},
	}

	svc, _, _, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, subs)
	notifMock.NotifyErrFor = map[string]error{"bob@example.com": errors.New("smtp refused")}

	okBefore := testutil.ToFloat64(metrics.RepoNotifyTotal.WithLabelValues("success"))
	errBefore := testutil.ToFloat64(metrics.RepoNotifyTotal.WithLabelValues("error"))
	if err := svc.CheckRepoTagAndAlert(context.Background(), repo); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	okAfter := testutil.ToFloat64(metrics.RepoNotifyTotal.WithLabelValues("success"))
	errAfter := testutil.ToFloat64(metrics.RepoNotifyTotal.WithLabelValues("error"))

	if got := okAfter - okBefore; got != 1 {
		t.Fatalf("expected 1 notify success increment, got %v", got)
	}
	if got := errAfter - errBefore; got != 1 {
		t.Fatalf("expected 1 notify error increment, got %v", got)
	}
}

func TestCheckAllReposTagAndAlert_EmptyRepositoryList_ReturnsNil(t *testing.T) {
	svc, _, _, _, _ := newTestService("v1.0.0", map[uint]*models.Repository{}, nil)

	if err := svc.CheckAllReposTagAndAlert(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCheckRepoTagAndAlert_VersionChanged_UpdatesRepoVersionInStore(t *testing.T) {
	repo := &models.Repository{ID: 1, Owner: "owner", Name: "repo", Version: "v1.0.0"}

	svc, repoMock, _, _, _ := newTestService("v3.0.0", map[uint]*models.Repository{1: repo}, []*models.Subscription{})

	if err := svc.CheckRepoTagAndAlert(context.Background(), repo); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stored := repoMock.Repositories[1]
	if stored.Version != "v3.0.0" {
		t.Fatalf("expected stored version v3.0.0, got %s", stored.Version)
	}
}
