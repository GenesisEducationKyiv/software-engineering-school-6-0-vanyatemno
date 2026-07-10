package notifications

import (
	"fmt"

	"ghnotify/contract"

	"se-school/internal/models"
)

func BuildConfirmEmailPayload(frontendURL, code string) contract.ConfirmEmailPayload {
	return contract.ConfirmEmailPayload{
		Code: code,
		Link: fmt.Sprintf("%s/confirm/%s", frontendURL, code),
	}
}

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
