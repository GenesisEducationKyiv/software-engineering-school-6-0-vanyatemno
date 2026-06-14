package notifications

import (
	"fmt"

	"ghnotify/contract"

	"se-school/internal/models"
)

// BuildConfirmEmailPayload assembles the confirmation-email payload, including
// the front-end confirmation link, for publishing to the notifications service.
func BuildConfirmEmailPayload(frontendURL, code string) contract.ConfirmEmailPayload {
	return contract.ConfirmEmailPayload{
		Code: code,
		Link: fmt.Sprintf("%s/confirm/%s", frontendURL, code),
	}
}

// BuildRepositoryUpdateEmailPayload assembles the release-notification payload,
// including the unsubscribe link, for publishing to the notifications service.
func BuildRepositoryUpdateEmailPayload(
	frontendURL string,
	repo *models.Repository,
) contract.RepositoryUpdateEmailPayload {
	return contract.RepositoryUpdateEmailPayload{
		Name:           repo.Name,
		Owner:          repo.Owner,
		Version:        repo.Version,
		UnsubscribeURL: fmt.Sprintf("%s/unsubscribe/%s", frontendURL, repo.Name),
	}
}
