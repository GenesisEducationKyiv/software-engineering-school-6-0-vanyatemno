package integration

import (
	"net/http"
	"testing"

	"se-school/internal/models/dto"
	"se-school/tests/integration/helpers"
)

func TestSuite_WiresServiceAndMSW(t *testing.T) {
	suite := helpers.NewSuite(t)

	if suite.Svc == nil {
		t.Fatal("expected subscription service to be wired, got nil")
	}

	suite.GH.Get("/repos/:owner/:repo/releases/latest", func(req helpers.Request) helpers.Response {
		return helpers.JSON(http.StatusOK, map[string]any{"tag_name": "v9.9.9"})
	})

	err := suite.Svc.Create(suite.Ctx, &dto.CreateSubscriptionRequest{
		Email: "smoke@example.com",
		Repo:  "octo/test",
	})
	if err != nil {
		t.Fatalf("expected smoke Create to succeed, got %v", err)
	}

	sub := suite.FindSubscriptionByEmail(t, "smoke@example.com")
	if sub.Repository == nil || sub.Repository.Version != "v9.9.9" {
		t.Fatalf("expected repository version v9.9.9, got %+v", sub.Repository)
	}
	if suite.GH.CallCount() != 1 {
		t.Fatalf("expected 1 msw call, got %d", suite.GH.CallCount())
	}
	if len(suite.Notifier.SendEmailCalls) != 1 {
		t.Fatalf("expected 1 confirmation email, got %d", len(suite.Notifier.SendEmailCalls))
	}
}
