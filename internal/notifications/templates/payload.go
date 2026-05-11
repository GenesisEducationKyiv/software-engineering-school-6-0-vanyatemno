package templates

import (
	"fmt"
	"se-school/internal/models"
)

func BuildConfirmEmailPayload(frontendURL, code string) ConfirmEmailPayload {
	return ConfirmEmailPayload{
		Code: code,
		Link: fmt.Sprintf(
			"%s/confirm/%s",
			frontendURL,
			code,
		),
	}
}

func BuildRepositoryUpdateEmailPayload(
	frontendURL string,
	repo *models.Repository,
) RepositoryUpdateEmailPayload {
	return RepositoryUpdateEmailPayload{
		Name:           repo.Name,
		Owner:          repo.Owner,
		Version:        repo.Version,
		UnsubscribeURL: fmt.Sprintf("%s/unsubscribe/%s", frontendURL, repo.Name),
	}
}
