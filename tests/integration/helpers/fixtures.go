package helpers

import (
	"testing"
	"time"

	"se-school/internal/models"
)

// SeedSubscription inserts a Repository, two Codes (confirm + unsubscribe)
// and a Subscription tying them together. Returns the persisted
// subscription with code values populated.
func (s *Suite) SeedSubscription(
	t *testing.T,
	email, owner, name, version string,
	isConfirmed bool,
) *models.Subscription {
	t.Helper()

	repo := &models.Repository{Owner: owner, Name: name, Version: version}
	if err := s.DB.Create(repo).Error; err != nil {
		t.Fatalf("seed repository: %v", err)
	}

	confirmCode := &models.Code{
		Code:      "CONF" + randSuffix(),
		Type:      models.CodeTypeConfirm,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	unsubCode := &models.Code{
		Code:      "UNSUB-" + randSuffix(),
		Type:      models.CodeTypeUnsubscribe,
		ExpiresAt: time.Now().Add(10 * 365 * 24 * time.Hour),
	}
	if err := s.DB.Create(confirmCode).Error; err != nil {
		t.Fatalf("seed confirm code: %v", err)
	}
	if err := s.DB.Create(unsubCode).Error; err != nil {
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
	if err := s.DB.Create(sub).Error; err != nil {
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
	var n int64
	if err := s.DB.Model(&models.Subscription{}).Count(&n).Error; err != nil {
		t.Fatalf("count subscriptions: %v", err)
	}
	return n
}

func (s *Suite) CountRepositories(t *testing.T) int64 {
	t.Helper()
	var n int64
	if err := s.DB.Model(&models.Repository{}).Count(&n).Error; err != nil {
		t.Fatalf("count repositories: %v", err)
	}
	return n
}

func (s *Suite) CountCodes(t *testing.T) int64 {
	t.Helper()
	var n int64
	if err := s.DB.Model(&models.Code{}).Count(&n).Error; err != nil {
		t.Fatalf("count codes: %v", err)
	}
	return n
}

func (s *Suite) FindSubscriptionByEmail(t *testing.T, email string) *models.Subscription {
	t.Helper()
	var sub models.Subscription
	if err := s.DB.
		Preload("SubscribeCode").
		Preload("UnsubscribeCode").
		Preload("Repository").
		Where("email = ?", email).
		First(&sub).Error; err != nil {
		t.Fatalf("find subscription %s: %v", email, err)
	}
	return &sub
}

func (s *Suite) CodeExists(t *testing.T, id uint) bool {
	t.Helper()
	var n int64
	if err := s.DB.Model(&models.Code{}).Where("id = ?", id).Count(&n).Error; err != nil {
		t.Fatalf("count code %d: %v", id, err)
	}
	return n > 0
}

var randCounter int64

func randSuffix() string {
	randCounter++
	return time.Now().Format("150405.000000") + "-" + itoa(randCounter)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
