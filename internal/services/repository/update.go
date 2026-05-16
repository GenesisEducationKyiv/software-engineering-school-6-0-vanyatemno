package repository

import (
	"context"
	"se-school/internal/models"
	"se-school/internal/notifications/templates"

	"go.uber.org/zap"
)

func (s *Service) CheckRepoTagAndAlert(ctx context.Context, repo *models.Repository) error {
	currentVersion, err := s.githubService.GetRepositoryVersion(ctx, repo.Owner, repo.Name)
	if err != nil {
		zap.L().Error("failed to fetch current repository version", zap.Error(err))
		return err
	}
	if currentVersion == repo.Version {
		return nil
	}
	repo, err = s.repositoriesRepository.UpdateTag(repo.ID, currentVersion)
	if err != nil {
		zap.L().Error("failed to update repository version", zap.Error(err))
		return err
	}
	err = s.sendRepositoryNotificationUpdates(repo)
	if err != nil {
		zap.L().Error("failed to send repository notification updates", zap.Error(err))
	}

	return nil
}

func (s *Service) sendRepositoryNotificationUpdates(repo *models.Repository) error {
	subs, err := s.subscriptionsRepository.GetUnupdated(repo.ID, repo.Version)
	zap.L().Debug("found unupdated subscriptions", zap.Int("subscriptions_count", len(subs)))
	if err != nil {
		return err
	}
	err = s.sendUpdates(repo, subs)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) sendUpdates(repo *models.Repository, subs []*models.Subscription) error {
	emails := make([]string, 0, len(subs))
	for _, sub := range subs {
		emails = append(emails, sub.Email)
	}
	payload := templates.BuildRepositoryUpdateEmailPayload(s.frontendURL, repo)

	zap.L().Debug(
		"sending repository update email payload",
		zap.Any("payload", payload),
		zap.Any("emails", emails),
	)
	err := s.notificationsService.SendEmail(emails, templates.RepositoryUpdated, payload)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) CheckAllReposTagAndAlert(ctx context.Context) error {
	repos, err := s.repositoriesRepository.GetAll()
	if err != nil {
		zap.L().Error("failed to fetch all repositories", zap.Error(err))
		return err
	}

	var errs []error
	for _, repo := range repos {
		err = s.CheckRepoTagAndAlert(ctx, repo)
		if err != nil {
			zap.L().Error("failed to update repository", zap.Error(err))
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		zap.L().Error("errors occurred on repositories update", zap.Int("errors_count", len(errs)))
		// todo: consolidate and return errors
	}

	return nil
}
