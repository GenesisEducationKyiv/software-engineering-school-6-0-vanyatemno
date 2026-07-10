package helpers

import (
	"testing"

	"se-school/internal/models"
)

// SeedRepository inserts a single repository row and returns it with its
// persisted ID populated.
func (s *Suite) SeedRepository(t *testing.T, owner, name, version string) *models.Repository {
	t.Helper()
	repo := &models.Repository{Owner: owner, Name: name, Version: version}
	if err := s.RepoRepo.Create(s.Ctx, repo); err != nil {
		t.Fatalf("seed repository: %v", err)
	}
	return repo
}

// SeedSubscription inserts a Repository, two Codes (confirm + unsubscribe)
// and a Subscription tying them together. Returns the persisted
// subscription with code values populated. Codes are built with the real
// code factory so their generated values are valid confirm/unsub tokens.
func (s *Suite) SeedSubscription(
	t *testing.T,
	email, owner, name, version string,
	isConfirmed bool,
) *models.Subscription {
	t.Helper()

	repo := s.SeedRepository(t, owner, name, version)

	confirmCode, err := s.Factory.New(models.CodeTypeConfirm)
	if err != nil {
		t.Fatalf("build confirm code: %v", err)
	}
	if err := s.CodeRepo.Create(s.Ctx, confirmCode); err != nil {
		t.Fatalf("seed confirm code: %v", err)
	}

	unsubCode, err := s.Factory.New(models.CodeTypeUnsubscribe)
	if err != nil {
		t.Fatalf("build unsubscribe code: %v", err)
	}
	if err := s.CodeRepo.Create(s.Ctx, unsubCode); err != nil {
		t.Fatalf("seed unsubscribe code: %v", err)
	}

	sub := &models.Subscription{
		RepositoryID:      repo.ID,
		SubscribeCodeID:   confirmCode.ID,
		UnsubscribeCodeID: unsubCode.ID,
		Email:             email,
		IsConfirmed:       isConfirmed,
		LastSeenTag:       version,
	}
	if err := s.SubRepo.Create(s.Ctx, sub); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}
	sub.SubscribeCode = confirmCode
	sub.UnsubscribeCode = unsubCode
	sub.Repository = repo
	return sub
}

// CountSubscriptions returns the number of non-soft-deleted subscription rows.
func (s *Suite) CountSubscriptions(t *testing.T) int64 {
	t.Helper()
	return s.countLive(t, "subscriptions")
}

func (s *Suite) CountRepositories(t *testing.T) int64 {
	t.Helper()
	return s.countLive(t, "repositories")
}

func (s *Suite) CountCodes(t *testing.T) int64 {
	t.Helper()
	return s.countLive(t, "codes")
}

// countLive counts rows in the given table that have not been soft-deleted.
// The table name is a fixed internal constant, never user input.
func (s *Suite) countLive(t *testing.T, table string) int64 {
	t.Helper()
	var n int64
	if err := s.DB.QueryRow(s.Ctx,
		"SELECT count(*) FROM "+table+" WHERE deleted_at IS NULL",
	).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// CountLiveSubscriptionsByID returns 1 if the subscription is still live
// (not soft-deleted), 0 otherwise.
func (s *Suite) CountLiveSubscriptionsByID(t *testing.T, id uint) int64 {
	t.Helper()
	var n int64
	if err := s.DB.QueryRow(s.Ctx,
		"SELECT count(*) FROM subscriptions WHERE id = $1 AND deleted_at IS NULL", id,
	).Scan(&n); err != nil {
		t.Fatalf("count subscription %d: %v", id, err)
	}
	return n
}

// FindSubscriptionByEmail returns the first active subscription for the email,
// with its Repository populated (reuses the production GetByEmail query).
func (s *Suite) FindSubscriptionByEmail(t *testing.T, email string) *models.Subscription {
	t.Helper()
	subs, err := s.SubRepo.GetByEmail(s.Ctx, email)
	if err != nil {
		t.Fatalf("find subscription %s: %v", email, err)
	}
	if len(subs) == 0 {
		t.Fatalf("no subscription found for %s", email)
	}
	return subs[0]
}

// CountOutbox returns the total number of outbox rows.
func (s *Suite) CountOutbox(t *testing.T) int64 {
	t.Helper()
	var n int64
	if err := s.DB.QueryRow(s.Ctx, "SELECT count(*) FROM outbox").Scan(&n); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	return n
}

// CountSagas returns the total number of saga_instances rows.
func (s *Suite) CountSagas(t *testing.T) int64 {
	t.Helper()
	var n int64
	if err := s.DB.QueryRow(s.Ctx, "SELECT count(*) FROM saga_instances").Scan(&n); err != nil {
		t.Fatalf("count sagas: %v", err)
	}
	return n
}

func (s *Suite) SagaState(t *testing.T, id string) string {
	t.Helper()
	var state string
	if err := s.DB.QueryRow(s.Ctx,
		"SELECT state FROM saga_instances WHERE id = $1", id,
	).Scan(&state); err != nil {
		t.Fatalf("saga state %s: %v", id, err)
	}
	return state
}

// CodeExists reports whether a non-soft-deleted code row with the given id exists.
func (s *Suite) CodeExists(t *testing.T, id uint) bool {
	t.Helper()
	var n int64
	if err := s.DB.QueryRow(s.Ctx,
		"SELECT count(*) FROM codes WHERE id = $1 AND deleted_at IS NULL", id,
	).Scan(&n); err != nil {
		t.Fatalf("count code %d: %v", id, err)
	}
	return n > 0
}
