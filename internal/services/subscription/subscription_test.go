package subscription

import (
	"context"
	"errors"
	"se-school/internal/config"
	"testing"

	"se-school/internal/integrations/github"
	"se-school/internal/models"
	"se-school/internal/models/dto"
	"se-school/internal/notifications"
	"se-school/internal/notifications/templates"
	codeRepo "se-school/internal/repositories/code"
	repoRepo "se-school/internal/repositories/repository"
	subRepo "se-school/internal/repositories/subscription"

	"gorm.io/gorm"
)

type testDeps struct {
	svc      *Service
	repos    *repoRepo.RepositoriesRepositoryMock
	subs     *subRepo.SubscriptionsRepositoryMock
	codes    *codeRepo.CodesRepositoryMock
	github   *github.GithubIntegrationMock
	notifier *notifications.NotificationsServiceMock
}

func setupTest() *testDeps {
	repos := repoRepo.NewRepositoriesRepositoryMock()
	subs := subRepo.NewSubscriptionsRepositoryMock()
	codes := codeRepo.NewCodesRepositoryMock()
	gh := github.NewGithubIntegrationMock("v1.0.0")
	notif := notifications.NewNotificationsServiceMock()

	svc := New(&config.Config{}, subs, repos, codes, gh, notif)

	return &testDeps{
		svc:      svc,
		repos:    repos,
		subs:     subs,
		codes:    codes,
		github:   gh,
		notifier: notif,
	}
}

func TestCreate_NewRepo_CreatesRepoAndSubscriptionAndSendsConfirmation(t *testing.T) {
	td := setupTest()

	td.codes.CreateResult = &models.Code{
		Model: gorm.Model{ID: 10},
		Code:  "ABC123",
		Type:  models.CodeTypeConfirm,
	}

	req := &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}

	err := td.svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(td.codes.CreateCalls) != 2 {
		t.Fatalf("expected 2 code Create calls (unsubscribe + confirm), got %d", len(td.codes.CreateCalls))
	}
	if td.codes.CreateCalls[0] != models.CodeTypeUnsubscribe {
		t.Fatalf("expected first code type %q, got %q", models.CodeTypeUnsubscribe, td.codes.CreateCalls[0])
	}
	if td.codes.CreateCalls[1] != models.CodeTypeConfirm {
		t.Fatalf("expected second code type %q, got %q", models.CodeTypeConfirm, td.codes.CreateCalls[1])
	}

	if len(td.notifier.SendEmailCalls) != 1 {
		t.Fatalf("expected 1 SendEmail call, got %d", len(td.notifier.SendEmailCalls))
	}

	call := td.notifier.SendEmailCalls[0]
	if len(call.Receivers) != 1 || call.Receivers[0] != "user@example.com" {
		t.Fatalf("expected receiver user@example.com, got %v", call.Receivers)
	}
	if call.Template != templates.Confirmation {
		t.Fatalf("expected template %q, got %q", templates.Confirmation, call.Template)
	}
}

func TestCreate_ExistingRepo_UsesExistingRepoWithoutGithubCall(t *testing.T) {
	td := setupTest()

	td.repos.Repositories[1] = &models.Repository{
		Model:   gorm.Model{ID: 1},
		Owner:   "owner",
		Name:    "repo",
		Version: "v1.0.0",
	}

	td.github.SetErrToReturn(errors.New("should not be called"))

	req := &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}

	err := td.svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(td.notifier.SendEmailCalls) != 1 {
		t.Fatalf("expected 1 SendEmail call, got %d", len(td.notifier.SendEmailCalls))
	}
}

func TestCreate_InvalidRepoFormat_ReturnsError(t *testing.T) {
	td := setupTest()

	req := &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "invalid-repo-format",
	}

	err := td.svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid repo format, got nil")
	}
}

func TestCreate_GithubError_ReturnsError(t *testing.T) {
	td := setupTest()
	td.github.SetErrToReturn(errors.New("github unavailable"))

	req := &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}

	err := td.svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "github unavailable" {
		t.Fatalf("expected 'github unavailable', got %q", err.Error())
	}
}

func TestCreate_CodeCreationError_ReturnsError(t *testing.T) {
	td := setupTest()
	td.codes.CreateErr = errors.New("code generation failed")

	req := &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}

	err := td.svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "code generation failed" {
		t.Fatalf("expected 'code generation failed', got %q", err.Error())
	}
}

func TestCreate_SubscriptionCreateError_ReturnsError(t *testing.T) {
	td := setupTest()
	td.subs.CreateErr = errors.New("duplicate subscription")

	req := &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}

	err := td.svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "duplicate subscription" {
		t.Fatalf("expected 'duplicate subscription', got %q", err.Error())
	}

	if len(td.notifier.SendEmailCalls) != 0 {
		t.Fatalf("expected no SendEmail calls after subscription creation failure, got %d", len(td.notifier.SendEmailCalls))
	}
}

func TestCreate_SendEmailError_ReturnsError(t *testing.T) {
	td := setupTest()
	td.notifier.SendEmailErr = errors.New("smtp failure")

	req := &dto.CreateSubscriptionRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}

	err := td.svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "smtp failure" {
		t.Fatalf("expected 'smtp failure', got %q", err.Error())
	}
}

func TestConfirm_ValidToken_SetsIsConfirmedAndDeletesCode(t *testing.T) {
	td := setupTest()

	td.codes.GetResult = &models.Code{
		Model: gorm.Model{ID: 5},
		Code:  "ABC123",
		Type:  models.CodeTypeConfirm,
	}
	td.subs.GetByCodeResult = &models.Subscription{
		Model:       gorm.Model{ID: 1},
		Email:       "user@example.com",
		IsConfirmed: false,
	}

	req := &dto.ConfirmSubscriptionRequest{Token: "ABC123"}

	err := td.svc.Confirm(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(td.codes.DeleteCalls) != 1 {
		t.Fatalf("expected 1 code Delete call, got %d", len(td.codes.DeleteCalls))
	}
	if td.codes.DeleteCalls[0] != 5 {
		t.Fatalf("expected code ID 5 to be deleted, got %d", td.codes.DeleteCalls[0])
	}
}

func TestConfirm_InvalidToken_ReturnsError(t *testing.T) {
	td := setupTest()
	td.codes.GetErr = errors.New("code not found")

	req := &dto.ConfirmSubscriptionRequest{Token: "INVALID"}

	err := td.svc.Confirm(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "code not found" {
		t.Fatalf("expected 'code not found', got %q", err.Error())
	}
}

func TestConfirm_SubscriptionNotFound_ReturnsError(t *testing.T) {
	td := setupTest()

	td.codes.GetResult = &models.Code{
		Model: gorm.Model{ID: 5},
		Code:  "ABC123",
		Type:  models.CodeTypeConfirm,
	}
	td.subs.GetByCodeErr = errors.New("subscription not found")

	req := &dto.ConfirmSubscriptionRequest{Token: "ABC123"}

	err := td.svc.Confirm(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "subscription not found" {
		t.Fatalf("expected 'subscription not found', got %q", err.Error())
	}
}

func TestConfirm_SaveError_ReturnsError(t *testing.T) {
	td := setupTest()

	td.codes.GetResult = &models.Code{
		Model: gorm.Model{ID: 5},
		Code:  "ABC123",
		Type:  models.CodeTypeConfirm,
	}
	td.subs.GetByCodeResult = &models.Subscription{
		Model: gorm.Model{ID: 1},
		Email: "user@example.com",
	}
	td.subs.SaveErr = errors.New("db save failed")

	req := &dto.ConfirmSubscriptionRequest{Token: "ABC123"}

	err := td.svc.Confirm(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db save failed" {
		t.Fatalf("expected 'db save failed', got %q", err.Error())
	}

	if len(td.codes.DeleteCalls) != 0 {
		t.Fatalf("expected no code Delete calls after save failure, got %d", len(td.codes.DeleteCalls))
	}
}

func TestUnsubscribe_ValidToken_DeletesSubscription(t *testing.T) {
	td := setupTest()

	td.codes.GetResult = &models.Code{
		Model: gorm.Model{ID: 7},
		Code:  "unsub-uuid",
		Type:  models.CodeTypeUnsubscribe,
	}
	td.subs.GetByCodeResult = &models.Subscription{
		Model: gorm.Model{ID: 2},
		Email: "user@example.com",
	}

	req := &dto.UnsubscribeRequest{Token: "unsub-uuid"}

	err := td.svc.Unsubscribe(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUnsubscribe_InvalidToken_ReturnsError(t *testing.T) {
	td := setupTest()
	td.codes.GetErr = errors.New("code not found")

	req := &dto.UnsubscribeRequest{Token: "INVALID"}

	err := td.svc.Unsubscribe(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "code not found" {
		t.Fatalf("expected 'code not found', got %q", err.Error())
	}
}

func TestUnsubscribe_SubscriptionNotFound_ReturnsError(t *testing.T) {
	td := setupTest()

	td.codes.GetResult = &models.Code{
		Model: gorm.Model{ID: 7},
		Code:  "unsub-uuid",
		Type:  models.CodeTypeUnsubscribe,
	}
	td.subs.GetByCodeErr = errors.New("subscription not found")

	req := &dto.UnsubscribeRequest{Token: "unsub-uuid"}

	err := td.svc.Unsubscribe(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUnsubscribe_DeleteError_ReturnsError(t *testing.T) {
	td := setupTest()

	td.codes.GetResult = &models.Code{
		Model: gorm.Model{ID: 7},
		Code:  "unsub-uuid",
		Type:  models.CodeTypeUnsubscribe,
	}
	td.subs.GetByCodeResult = &models.Subscription{
		Model: gorm.Model{ID: 2},
		Email: "user@example.com",
	}
	td.subs.DeleteErr = errors.New("delete failed")

	req := &dto.UnsubscribeRequest{Token: "unsub-uuid"}

	err := td.svc.Unsubscribe(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "delete failed" {
		t.Fatalf("expected 'delete failed', got %q", err.Error())
	}
}

func TestListByEmail_ReturnsSubscriptions(t *testing.T) {
	td := setupTest()

	td.subs.GetByEmailResult = []*models.Subscription{
		{Model: gorm.Model{ID: 1}, Email: "user@example.com"},
		{Model: gorm.Model{ID: 2}, Email: "user@example.com"},
	}

	req := &dto.GetSubscriptionsRequest{Email: "user@example.com"}

	result, err := td.svc.ListByEmail(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(result))
	}
}

func TestListByEmail_MapsRepositoryField(t *testing.T) {
	td := setupTest()

	td.subs.GetByEmailResult = []*models.Subscription{
		{
			Model: gorm.Model{ID: 1},
			Email: "user@example.com",
			Repository: &models.Repository{
				Owner: "golang",
				Name:  "go",
			},
			IsConfirmed: true,
			LastSeenTag: "v1.22.0",
		},
		{
			Model: gorm.Model{ID: 2},
			Email: "user@example.com",
		},
	}

	req := &dto.GetSubscriptionsRequest{Email: "user@example.com"}

	result, err := td.svc.ListByEmail(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result[0].Repo != "golang/go" {
		t.Fatalf("expected repo %q, got %q", "golang/go", result[0].Repo)
	}
	if !result[0].Confirmed {
		t.Fatal("expected first subscription to be confirmed")
	}
	if result[0].LastSeenTag != "v1.22.0" {
		t.Fatalf("expected last_seen_tag %q, got %q", "v1.22.0", result[0].LastSeenTag)
	}
	if result[1].Repo != "" {
		t.Fatalf("expected empty repo for nil Repository, got %q", result[1].Repo)
	}
}

func TestListByEmail_Error_ReturnsError(t *testing.T) {
	td := setupTest()
	td.subs.GetByEmailErr = errors.New("db query failed")

	req := &dto.GetSubscriptionsRequest{Email: "user@example.com"}

	result, err := td.svc.ListByEmail(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Fatalf("expected nil result, got %v", result)
	}
}

func TestListByEmail_NoSubscriptions_ReturnsEmptySlice(t *testing.T) {
	td := setupTest()
	td.subs.GetByEmailResult = []*models.Subscription{}

	req := &dto.GetSubscriptionsRequest{Email: "nobody@example.com"}

	result, err := td.svc.ListByEmail(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 subscriptions, got %d", len(result))
	}
}

func TestCreate_RepoFormatVariants(t *testing.T) {
	testCases := []struct {
		name      string
		repo      string
		expectErr bool
	}{
		{name: "valid owner/repo", repo: "owner/repo", expectErr: false},
		{name: "missing slash", repo: "ownerrepo", expectErr: true},
		{name: "too many slashes", repo: "owner/repo/extra", expectErr: true},
		{name: "empty string", repo: "", expectErr: true},
		{name: "only slash", repo: "/", expectErr: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			td := setupTest()

			req := &dto.CreateSubscriptionRequest{
				Email: "user@example.com",
				Repo:  tc.repo,
			}

			err := td.svc.Create(context.Background(), req)
			if tc.expectErr && err == nil {
				t.Fatalf("expected error for repo %q, got nil", tc.repo)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("expected no error for repo %q, got %v", tc.repo, err)
			}
		})
	}
}
