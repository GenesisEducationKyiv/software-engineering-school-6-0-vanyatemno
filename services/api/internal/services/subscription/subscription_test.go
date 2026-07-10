package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"ghnotify/contract"

	"se-school/internal/integrations/github"
	"se-school/internal/models"
	"se-school/internal/models/dto"
	"se-school/internal/models/factories/codes"
	codeRepo "se-school/internal/repositories/code"
	outboxRepo "se-school/internal/repositories/outbox"
	repoRepo "se-school/internal/repositories/repository"
	sagaRepo "se-school/internal/repositories/saga"
	subRepo "se-school/internal/repositories/subscription"
)

// fakeUnitOfWork runs the orchestrator's transactional closure directly against
// the in-memory mocks — no real transaction, but the same repository calls are
// recorded so tests can assert on them.
type fakeUnitOfWork struct {
	repos *TxRepos
	err   error // if set, Do returns it without running fn (simulates a tx failure)
}

func (u *fakeUnitOfWork) Do(ctx context.Context, fn func(ctx context.Context, r *TxRepos) error) error {
	if u.err != nil {
		return u.err
	}
	return fn(ctx, u.repos)
}

type testDeps struct {
	svc     *Service
	repos   *repoRepo.RepositoriesRepositoryMock
	subs    *subRepo.SubscriptionsRepositoryMock
	codes   *codeRepo.CodesRepositoryMock
	factory *codes.FactoryMock
	github  *github.GithubIntegrationMock
	sagas   *sagaRepo.RepositoryMock
	outbox  *outboxRepo.RepositoryMock
	uow     *fakeUnitOfWork
}

func setupTest() *testDeps {
	repos := repoRepo.NewRepositoriesRepositoryMock()
	subs := subRepo.NewSubscriptionsRepositoryMock()
	codesRepo := codeRepo.NewCodesRepositoryMock()
	factory := codes.NewFactoryMock()
	gh := github.NewGithubIntegrationMock("v1.0.0")
	sagas := sagaRepo.NewRepositoryMock()
	ob := outboxRepo.NewRepositoryMock()

	uow := &fakeUnitOfWork{repos: &TxRepos{
		Subscriptions: subs,
		Codes:         codesRepo,
		Sagas:         sagas,
		Outbox:        ob,
	}}

	svc := New("http://frontend", subs, repos, codesRepo, factory, gh, uow, sagas, time.Minute)

	return &testDeps{
		svc:     svc,
		repos:   repos,
		subs:    subs,
		codes:   codesRepo,
		factory: factory,
		github:  gh,
		sagas:   sagas,
		outbox:  ob,
		uow:     uow,
	}
}

func TestCreate_NewRepo_PersistsSagaAndOutboxCommand(t *testing.T) {
	td := setupTest()

	req := &dto.CreateSubscriptionRequest{Email: "user@example.com", Repo: "owner/repo"}

	sagaID, err := td.svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sagaID == "" {
		t.Fatal("expected a non-empty saga id")
	}

	if len(td.factory.NewCalls) != 2 {
		t.Fatalf("expected 2 factory New calls, got %d", len(td.factory.NewCalls))
	}
	if td.factory.NewCalls[0] != models.CodeTypeUnsubscribe || td.factory.NewCalls[1] != models.CodeTypeConfirm {
		t.Fatalf("unexpected code type order: %v", td.factory.NewCalls)
	}
	if len(td.codes.CreateCalls) != 2 {
		t.Fatalf("expected 2 code Create calls, got %d", len(td.codes.CreateCalls))
	}

	// A saga instance is created in AWAITING_NOTIFICATION with the returned id.
	if len(td.sagas.Created) != 1 {
		t.Fatalf("expected 1 saga created, got %d", len(td.sagas.Created))
	}
	saga := td.sagas.Created[0]
	if saga.ID != sagaID {
		t.Fatalf("saga id %q != returned %q", saga.ID, sagaID)
	}
	if saga.State != models.SagaStateAwaitingNotification {
		t.Fatalf("expected saga state %q, got %q", models.SagaStateAwaitingNotification, saga.State)
	}

	// Exactly one outbox command, carrying a Confirmation message for the saga.
	if len(td.outbox.Created) != 1 {
		t.Fatalf("expected 1 outbox message, got %d", len(td.outbox.Created))
	}
	msg := td.outbox.Created[0]
	if msg.Topic != contract.Topic {
		t.Fatalf("expected outbox topic %q, got %q", contract.Topic, msg.Topic)
	}
	var decoded contract.Message
	if err := json.Unmarshal(msg.Payload, &decoded); err != nil {
		t.Fatalf("failed to decode outbox payload: %v", err)
	}
	if decoded.Template != contract.Confirmation {
		t.Fatalf("expected template %q, got %q", contract.Confirmation, decoded.Template)
	}
	if decoded.SagaID != sagaID {
		t.Fatalf("expected message saga id %q, got %q", sagaID, decoded.SagaID)
	}
	if len(decoded.Receivers) != 1 || decoded.Receivers[0] != "user@example.com" {
		t.Fatalf("unexpected receivers: %v", decoded.Receivers)
	}
	if msg.KafkaKey != decoded.IdempotencyKey || decoded.IdempotencyKey == "" {
		t.Fatalf("expected outbox key to equal the message idempotency key")
	}
}

func TestCreate_ExistingRepo_UsesExistingRepoWithoutGithubCall(t *testing.T) {
	td := setupTest()

	td.repos.Repositories[1] = &models.Repository{ID: 1, Owner: "owner", Name: "repo", Version: "v1.0.0"}
	td.github.SetErrToReturn(errors.New("should not be called"))

	req := &dto.CreateSubscriptionRequest{Email: "user@example.com", Repo: "owner/repo"}

	sagaID, err := td.svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sagaID == "" {
		t.Fatal("expected a non-empty saga id")
	}
	if len(td.outbox.Created) != 1 {
		t.Fatalf("expected 1 outbox message, got %d", len(td.outbox.Created))
	}
}

func TestCreate_InvalidRepoFormat_ReturnsError(t *testing.T) {
	td := setupTest()

	_, err := td.svc.Create(context.Background(), &dto.CreateSubscriptionRequest{
		Email: "user@example.com", Repo: "invalid-repo-format",
	})
	if err == nil {
		t.Fatal("expected error for invalid repo format, got nil")
	}
}

func TestCreate_GithubError_ReturnsError(t *testing.T) {
	td := setupTest()
	td.github.SetErrToReturn(errors.New("github unavailable"))

	_, err := td.svc.Create(context.Background(), &dto.CreateSubscriptionRequest{
		Email: "user@example.com", Repo: "owner/repo",
	})
	if err == nil || err.Error() != "github unavailable" {
		t.Fatalf("expected 'github unavailable', got %v", err)
	}
}

func TestCreate_CodeCreationError_ReturnsError(t *testing.T) {
	td := setupTest()
	td.factory.NewErr = errors.New("code generation failed")

	_, err := td.svc.Create(context.Background(), &dto.CreateSubscriptionRequest{
		Email: "user@example.com", Repo: "owner/repo",
	})
	if err == nil || err.Error() != "code generation failed" {
		t.Fatalf("expected 'code generation failed', got %v", err)
	}
}

func TestCreate_SubscriptionCreateError_RollsBackViaTx(t *testing.T) {
	td := setupTest()
	td.subs.CreateErr = errors.New("duplicate subscription")

	_, err := td.svc.Create(context.Background(), &dto.CreateSubscriptionRequest{
		Email: "user@example.com", Repo: "owner/repo",
	})
	if err == nil || err.Error() != "duplicate subscription" {
		t.Fatalf("expected 'duplicate subscription', got %v", err)
	}

	// The closure aborts before persisting the outbox command, so no command is
	// emitted for a subscription that never committed.
	if len(td.outbox.Created) != 0 {
		t.Fatalf("expected no outbox command after subscription failure, got %d", len(td.outbox.Created))
	}
}

func TestCreate_TxError_ReturnsError(t *testing.T) {
	td := setupTest()
	td.uow.err = errors.New("tx commit failed")

	_, err := td.svc.Create(context.Background(), &dto.CreateSubscriptionRequest{
		Email: "user@example.com", Repo: "owner/repo",
	})
	if err == nil || err.Error() != "tx commit failed" {
		t.Fatalf("expected 'tx commit failed', got %v", err)
	}
}

func TestConfirm_ValidToken_SetsIsConfirmedAndDeletesCode(t *testing.T) {
	td := setupTest()

	td.codes.GetResult = &models.Code{ID: 5, Code: "ABC123", Type: models.CodeTypeConfirm}
	td.subs.GetByCodeResult = &models.Subscription{ID: 1, Email: "user@example.com", IsConfirmed: false}

	err := td.svc.Confirm(context.Background(), &dto.ConfirmSubscriptionRequest{Token: "ABC123"})
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

	err := td.svc.Confirm(context.Background(), &dto.ConfirmSubscriptionRequest{Token: "INVALID"})
	if err == nil || err.Error() != "code not found" {
		t.Fatalf("expected 'code not found', got %v", err)
	}
}

func TestConfirm_SubscriptionNotFound_ReturnsError(t *testing.T) {
	td := setupTest()

	td.codes.GetResult = &models.Code{ID: 5, Code: "ABC123", Type: models.CodeTypeConfirm}
	td.subs.GetByCodeErr = errors.New("subscription not found")

	err := td.svc.Confirm(context.Background(), &dto.ConfirmSubscriptionRequest{Token: "ABC123"})
	if err == nil || err.Error() != "subscription not found" {
		t.Fatalf("expected 'subscription not found', got %v", err)
	}
}

func TestConfirm_SaveError_ReturnsError(t *testing.T) {
	td := setupTest()

	td.codes.GetResult = &models.Code{ID: 5, Code: "ABC123", Type: models.CodeTypeConfirm}
	td.subs.GetByCodeResult = &models.Subscription{ID: 1, Email: "user@example.com"}
	td.subs.SaveErr = errors.New("db save failed")

	err := td.svc.Confirm(context.Background(), &dto.ConfirmSubscriptionRequest{Token: "ABC123"})
	if err == nil || err.Error() != "db save failed" {
		t.Fatalf("expected 'db save failed', got %v", err)
	}

	if len(td.codes.DeleteCalls) != 0 {
		t.Fatalf("expected no code Delete calls after save failure, got %d", len(td.codes.DeleteCalls))
	}
}

func TestUnsubscribe_ValidToken_DeletesSubscription(t *testing.T) {
	td := setupTest()

	td.codes.GetResult = &models.Code{ID: 7, Code: "unsub-uuid", Type: models.CodeTypeUnsubscribe}
	td.subs.GetByCodeResult = &models.Subscription{ID: 2, Email: "user@example.com"}

	err := td.svc.Unsubscribe(context.Background(), &dto.UnsubscribeRequest{Token: "unsub-uuid"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUnsubscribe_InvalidToken_ReturnsError(t *testing.T) {
	td := setupTest()
	td.codes.GetErr = errors.New("code not found")

	err := td.svc.Unsubscribe(context.Background(), &dto.UnsubscribeRequest{Token: "INVALID"})
	if err == nil || err.Error() != "code not found" {
		t.Fatalf("expected 'code not found', got %v", err)
	}
}

func TestUnsubscribe_SubscriptionNotFound_ReturnsError(t *testing.T) {
	td := setupTest()

	td.codes.GetResult = &models.Code{ID: 7, Code: "unsub-uuid", Type: models.CodeTypeUnsubscribe}
	td.subs.GetByCodeErr = errors.New("subscription not found")

	err := td.svc.Unsubscribe(context.Background(), &dto.UnsubscribeRequest{Token: "unsub-uuid"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUnsubscribe_DeleteError_ReturnsError(t *testing.T) {
	td := setupTest()

	td.codes.GetResult = &models.Code{ID: 7, Code: "unsub-uuid", Type: models.CodeTypeUnsubscribe}
	td.subs.GetByCodeResult = &models.Subscription{ID: 2, Email: "user@example.com"}
	td.subs.DeleteErr = errors.New("delete failed")

	err := td.svc.Unsubscribe(context.Background(), &dto.UnsubscribeRequest{Token: "unsub-uuid"})
	if err == nil || err.Error() != "delete failed" {
		t.Fatalf("expected 'delete failed', got %v", err)
	}
}

func TestListByEmail_ReturnsSubscriptions(t *testing.T) {
	td := setupTest()

	td.subs.GetByEmailResult = []*models.Subscription{
		{ID: 1, Email: "user@example.com"},
		{ID: 2, Email: "user@example.com"},
	}

	result, err := td.svc.ListByEmail(context.Background(), &dto.GetSubscriptionsRequest{Email: "user@example.com"})
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
			ID:          1,
			Email:       "user@example.com",
			Repository:  &models.Repository{Owner: "golang", Name: "go"},
			IsConfirmed: true,
			LastSeenTag: "v1.22.0",
		},
		{ID: 2, Email: "user@example.com"},
	}

	result, err := td.svc.ListByEmail(context.Background(), &dto.GetSubscriptionsRequest{Email: "user@example.com"})
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

	result, err := td.svc.ListByEmail(context.Background(), &dto.GetSubscriptionsRequest{Email: "user@example.com"})
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

	result, err := td.svc.ListByEmail(context.Background(), &dto.GetSubscriptionsRequest{Email: "nobody@example.com"})
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

			_, err := td.svc.Create(context.Background(), &dto.CreateSubscriptionRequest{
				Email: "user@example.com",
				Repo:  tc.repo,
			})
			if tc.expectErr && err == nil {
				t.Fatalf("expected error for repo %q, got nil", tc.repo)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("expected no error for repo %q, got %v", tc.repo, err)
			}
		})
	}
}
