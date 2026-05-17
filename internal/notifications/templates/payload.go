package templates

import (
	"fmt"
	"se-school/internal/config"
	"se-school/internal/models"
)

func BuildConfirmEmailPayload(cfg *config.Config, code string) ConfirmEmailPayload {
	return ConfirmEmailPayload{
		Code: code,
		Link: fmt.Sprintf(
			"%s/confirm/%s",
			cfg.FrontendURL,
			code,
		),
	}
}

func BuildRepositoryUpdateEmailPayload(
	cfg *config.Config,
	repo *models.Repository,
) RepositoryUpdateEmailPayload {
	return RepositoryUpdateEmailPayload{
		Name:           repo.Name,
		Owner:          repo.Owner,
		Version:        repo.Version,
		UnsubscribeURL: fmt.Sprintf("%s/unsubscribe/%s", cfg.FrontendURL, repo.Name),
	}
}
