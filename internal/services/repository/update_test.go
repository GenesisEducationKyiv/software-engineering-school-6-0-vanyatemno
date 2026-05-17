package repository

import (
	"context"
	"errors"
	"se-school/internal/config"
	"testing"

	"se-school/internal/integrations/github"
	"se-school/internal/models"
	"se-school/internal/notifications"
	"se-school/internal/notifications/templates"
	repoRepo "se-school/internal/repositories/repository"
	subRepo "se-school/internal/repositories/subscription"
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

	svc := New(&config.Config{}, repoMock, subMock, notifMock, githubMock)
	return svc, repoMock, subMock, notifMock, githubMock
}

func TestCheckRepoTagAndAlert_VersionUnchanged_SkipsUpdateAndNotification(t *testing.T) {
	repo := &models.Repository{
		ID:      1,
		Owner:   "owner",
		Name:    "repo",
		Version: "v1.0.0",
	}

	svc, repoMock, _, notifMock, _ := newTestService("v1.0.0", map[uint]*models.Repository{1: repo}, nil)

	err := svc.CheckRepoTagAndAlert(context.Background(), repo)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repoMock.UpdateTagCalls) != 0 {
		t.Fatalf("expected no UpdateTag calls, got %d", len(repoMock.UpdateTagCalls))
	}

	if len(notifMock.SendEmailCalls) != 0 {
		t.Fatalf("expected no SendEmail calls, got %d", len(notifMock.SendEmailCalls))
	}
}

func TestCheckRepoTagAndAlert_VersionChanged_UpdatesTagAndSendsNotification(t *testing.T) {
	repo := &models.Repository{
		ID:      1,
		Owner:   "owner",
		Name:    "repo",
		Version: "v1.0.0",
	}

	subs := []*models.Subscription{
		{Email: "alice@example.com"},
		{Email: "bob@example.com"},
	}

	svc, repoMock, _, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, subs)

	err := svc.CheckRepoTagAndAlert(context.Background(), repo)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repoMock.UpdateTagCalls) != 1 {
		t.Fatalf("expected 1 UpdateTag call, got %d", len(repoMock.UpdateTagCalls))
	}
	if repoMock.UpdateTagCalls[0].ID != 1 || repoMock.UpdateTagCalls[0].Tag != "v2.0.0" {
		t.Fatalf("expected UpdateTag(1, v2.0.0), got UpdateTag(%d, %s)",
			repoMock.UpdateTagCalls[0].ID, repoMock.UpdateTagCalls[0].Tag)
	}

	if len(notifMock.SendEmailCalls) != 1 {
		t.Fatalf("expected 1 SendEmail call, got %d", len(notifMock.SendEmailCalls))
	}

	call := notifMock.SendEmailCalls[0]
	if len(call.Receivers) != 2 {
		t.Fatalf("expected 2 receivers, got %d", len(call.Receivers))
	}
	if call.Template != templates.RepositoryUpdated {
		t.Fatalf("expected template %q, got %q", templates.RepositoryUpdated, call.Template)
	}

	payload, ok := call.Data.(templates.RepositoryUpdateEmailPayload)
	if !ok {
		t.Fatalf("expected RepositoryUpdateEmailPayload, got %T", call.Data)
	}
	if payload.Owner != "owner" || payload.Name != "repo" || payload.Version != "v2.0.0" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestCheckRepoTagAndAlert_GithubError_ReturnsError(t *testing.T) {
	repo := &models.Repository{
		ID:      1,
		Owner:   "owner",
		Name:    "repo",
		Version: "v1.0.0",
	}

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
	repo := &models.Repository{
		ID:      1,
		Owner:   "owner",
		Name:    "repo",
		Version: "v1.0.0",
	}

	svc, repoMock, _, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, nil)
	repoMock.UpdateTagErr = errors.New("db write failed")

	err := svc.CheckRepoTagAndAlert(context.Background(), repo)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db write failed" {
		t.Fatalf("expected 'db write failed', got %q", err.Error())
	}

	if len(notifMock.SendEmailCalls) != 0 {
		t.Fatalf("expected no SendEmail calls after UpdateTag failure, got %d", len(notifMock.SendEmailCalls))
	}
}

func TestCheckRepoTagAndAlert_GetUnupdatedError_ReturnsNil(t *testing.T) {
	repo := &models.Repository{
		ID:      1,
		Owner:   "owner",
		Name:    "repo",
		Version: "v1.0.0",
	}

	svc, _, subMock, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, nil)
	subMock.GetUnupdatedErr = errors.New("subscription query failed")

	err := svc.CheckRepoTagAndAlert(context.Background(), repo)
	if err != nil {
		t.Fatalf("expected nil (error is logged, not returned), got %v", err)
	}

	if len(notifMock.SendEmailCalls) != 0 {
		t.Fatalf("expected no SendEmail calls after GetUnupdated failure, got %d", len(notifMock.SendEmailCalls))
	}
}

func TestCheckRepoTagAndAlert_SendEmailError_ReturnsNil(t *testing.T) {
	repo := &models.Repository{
		ID:      1,
		Owner:   "owner",
		Name:    "repo",
		Version: "v1.0.0",
	}

	subs := []*models.Subscription{
		{Email: "alice@example.com"},
	}

	svc, _, _, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, subs)
	notifMock.SendEmailErr = errors.New("smtp failure")

	err := svc.CheckRepoTagAndAlert(context.Background(), repo)
	if err != nil {
		t.Fatalf("expected nil (error is logged, not returned), got %v", err)
	}
}

func TestCheckRepoTagAndAlert_NoSubscribers_SendsToEmptyList(t *testing.T) {
	repo := &models.Repository{
		ID:      1,
		Owner:   "owner",
		Name:    "repo",
		Version: "v1.0.0",
	}

	svc, _, _, notifMock, _ := newTestService("v2.0.0", map[uint]*models.Repository{1: repo}, []*models.Subscription{})

	err := svc.CheckRepoTagAndAlert(context.Background(), repo)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(notifMock.SendEmailCalls) != 1 {
		t.Fatalf("expected 1 SendEmail call even with empty subscriber list, got %d", len(notifMock.SendEmailCalls))
	}

	if len(notifMock.SendEmailCalls[0].Receivers) != 0 {
		t.Fatalf("expected 0 receivers, got %d", len(notifMock.SendEmailCalls[0].Receivers))
	}
}

func TestCheckAllReposTagAndAlert_ProcessesAllRepositories(t *testing.T) {
	repos := map[uint]*models.Repository{
		1: {ID: 1, Owner: "owner1", Name: "repo1", Version: "v1.0.0"},
		2: {ID: 2, Owner: "owner2", Name: "repo2", Version: "v1.0.0"},
	}

	svc, repoMock, _, _, _ := newTestService("v2.0.0", repos, []*models.Subscription{})

	err := svc.CheckAllReposTagAndAlert(context.Background())
	if err != nil {
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

	err := svc.CheckAllReposTagAndAlert(context.Background())
	if err != nil {
		t.Fatalf("expected nil (errors are collected but not returned), got %v", err)
	}
}

func TestCheckAllReposTagAndAlert_EmptyRepositoryList_ReturnsNil(t *testing.T) {
	svc, _, _, _, _ := newTestService("v1.0.0", map[uint]*models.Repository{}, nil)

	err := svc.CheckAllReposTagAndAlert(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCheckRepoTagAndAlert_VersionChanged_UpdatesRepoVersionInStore(t *testing.T) {
	repo := &models.Repository{
		ID:      1,
		Owner:   "owner",
		Name:    "repo",
		Version: "v1.0.0",
	}

	svc, repoMock, _, _, _ := newTestService("v3.0.0", map[uint]*models.Repository{1: repo}, []*models.Subscription{})

	err := svc.CheckRepoTagAndAlert(context.Background(), repo)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stored := repoMock.Repositories[1]
	if stored.Version != "v3.0.0" {
		t.Fatalf("expected stored version v3.0.0, got %s", stored.Version)
	}
}
