package integration

import (
	"testing"

	"se-school/internal/models/dto"
	"se-school/tests/integration/helpers"
)

func TestList_MultipleSubscriptions_ReturnsAllWithRepository(t *testing.T) {
	s := helpers.NewSuite(t)

	s.SeedSubscription(t, "user@example.com", "octocat", "hello-world", "v1.0.0", true)
	s.SeedSubscription(t, "user@example.com", "torvalds", "linux", "v6.10", false)

	subs, err := s.Svc.ListByEmail(&dto.GetSubscriptionsRequest{Email: "user@example.com"})
	if err != nil {
		t.Fatalf("ListByEmail: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(subs))
	}

	for _, sub := range subs {
		if sub.Repository == nil {
			t.Fatalf("expected Repository to be preloaded for sub %d", sub.ID)
		}
		if sub.Repository.Owner == "" || sub.Repository.Name == "" {
			t.Fatalf("expected Repository fields populated, got %+v", sub.Repository)
		}
	}
}

func TestList_NoSubscriptions_ReturnsEmptySliceNoError(t *testing.T) {
	s := helpers.NewSuite(t)

	subs, err := s.Svc.ListByEmail(&dto.GetSubscriptionsRequest{Email: "ghost@example.com"})
	if err != nil {
		t.Fatalf("ListByEmail with no matches must not error, got %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("expected 0 subscriptions, got %d", len(subs))
	}
}

func TestList_FiltersByEmail_DoesNotLeakOtherUsersSubscriptions(t *testing.T) {
	s := helpers.NewSuite(t)

	s.SeedSubscription(t, "alice@example.com", "octocat", "hello-world", "v1.0.0", true)
	s.SeedSubscription(t, "bob@example.com", "torvalds", "linux", "v6.10", true)
	s.SeedSubscription(t, "bob@example.com", "golang", "go", "go1.23", false)

	alice, err := s.Svc.ListByEmail(&dto.GetSubscriptionsRequest{Email: "alice@example.com"})
	if err != nil {
		t.Fatalf("ListByEmail alice: %v", err)
	}
	if len(alice) != 1 {
		t.Fatalf("expected 1 alice subscription, got %d", len(alice))
	}
	if alice[0].Email != "alice@example.com" {
		t.Fatalf("expected alice's row, got email %q", alice[0].Email)
	}

	bob, err := s.Svc.ListByEmail(&dto.GetSubscriptionsRequest{Email: "bob@example.com"})
	if err != nil {
		t.Fatalf("ListByEmail bob: %v", err)
	}
	if len(bob) != 2 {
		t.Fatalf("expected 2 bob subscriptions, got %d", len(bob))
	}
	for _, sub := range bob {
		if sub.Email != "bob@example.com" {
			t.Fatalf("expected only bob's rows, got email %q", sub.Email)
		}
	}
}

func TestList_IncludesConfirmedAndUnconfirmed(t *testing.T) {
	s := helpers.NewSuite(t)

	s.SeedSubscription(t, "user@example.com", "octocat", "hello-world", "v1.0.0", true)
	s.SeedSubscription(t, "user@example.com", "torvalds", "linux", "v6.10", false)

	subs, err := s.Svc.ListByEmail(&dto.GetSubscriptionsRequest{Email: "user@example.com"})
	if err != nil {
		t.Fatalf("ListByEmail: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected both confirmed + unconfirmed rows, got %d", len(subs))
	}

	var sawConfirmed, sawUnconfirmed bool
	for _, sub := range subs {
		if sub.IsConfirmed {
			sawConfirmed = true
		} else {
			sawUnconfirmed = true
		}
	}
	if !sawConfirmed || !sawUnconfirmed {
		t.Fatalf("expected one confirmed + one unconfirmed, got confirmed=%v unconfirmed=%v", sawConfirmed, sawUnconfirmed)
	}
}

func TestList_ExcludesSoftDeletedSubscriptions(t *testing.T) {
	s := helpers.NewSuite(t)

	sub := s.SeedSubscription(t, "user@example.com", "octocat", "hello-world", "v1.0.0", true)
	s.SeedSubscription(t, "user@example.com", "torvalds", "linux", "v6.10", true)

	if err := s.Svc.Unsubscribe(&dto.UnsubscribeRequest{Token: sub.UnsubscribeCode.Code}); err != nil {
		t.Fatalf("seed unsubscribe: %v", err)
	}

	subs, err := s.Svc.ListByEmail(&dto.GetSubscriptionsRequest{Email: "user@example.com"})
	if err != nil {
		t.Fatalf("ListByEmail: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected soft-deleted row excluded, got %d rows", len(subs))
	}
	if subs[0].ID == sub.ID {
		t.Fatalf("expected remaining row to differ from unsubscribed one, got %d", subs[0].ID)
	}
}
