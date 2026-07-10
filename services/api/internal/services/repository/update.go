package repository

import (
	"context"
	"fmt"

	"ghnotify/contract"

	"se-school/internal/metrics"
	"se-school/internal/models"
	"se-school/internal/notifications"

	"go.uber.org/zap"
)

func (s *Service) CheckRepoTagAndAlert(ctx context.Context, repo *models.Repository) error {
	currentVersion, err := s.githubService.GetRepositoryVersion(ctx, repo.Owner, repo.Name)
	if err != nil {
		zap.L().Error("failed to fetch current repository version", zap.Error(err))
		return err
	}

	// Persist the newest observed version. repo.Version is now only a record of
	// the latest tag: repeat-suppression rides on each subscription's
	// last_seen_tag (see sendUpdates), so a subscriber whose alert failed to
	// deliver is retried on the next run even though the repo version is current.
	if currentVersion != repo.Version {
		repo, err = s.repositoriesRepository.UpdateTag(ctx, repo.ID, currentVersion)
		if err != nil {
			zap.L().Error("failed to update repository version", zap.Error(err))
			return err
		}
	}

	// Reconcile on every run: notify any subscription whose last_seen_tag lags
	// the current version — a fresh release, or a prior alert that failed.
	if err := s.sendRepositoryNotificationUpdates(ctx, repo, currentVersion); err != nil {
		zap.L().Error("failed to send repository notification updates", zap.Error(err))
	}

	return nil
}

func (s *Service) sendRepositoryNotificationUpdates(ctx context.Context, repo *models.Repository, currentVersion string) error {
	subs, err := s.subscriptionsRepository.GetUnupdated(ctx, repo.ID, currentVersion)
	zap.L().Debug("found unupdated subscriptions", zap.Int("subscriptions_count", len(subs)))
	if err != nil {
		return err
	}

	return s.sendUpdates(ctx, repo, currentVersion, subs)
}

// sendUpdates notifies each subscriber synchronously over gRPC and, only on
// confirmed delivery, advances that subscription's last_seen_tag to
// currentVersion. A failed delivery leaves the tag unchanged so GetUnupdated
// returns the subscriber again on the next cron run; failures are isolated so one
// unreachable recipient does not stop the rest.
func (s *Service) sendUpdates(
	ctx context.Context,
	repo *models.Repository,
	currentVersion string,
	subs []*models.Subscription,
) error {
	payload := notifications.BuildRepositoryUpdateEmailPayload(s.frontendURL, repo)

	var failed int
	for _, sub := range subs {
		if err := s.notificationsService.Notify(ctx, sub.Email, contract.RepositoryUpdated, payload); err != nil {
			metrics.RepoNotifyTotal.WithLabelValues("error").Inc()
			zap.L().Error("failed to notify subscriber; leaving last_seen_tag for retry",
				zap.String("email", sub.Email), zap.Error(err))
			failed++
			continue
		}

		if err := s.subscriptionsRepository.UpdateLastSeenTag(ctx, sub.ID, currentVersion); err != nil {
			// Delivered, but we could not record it; the subscriber may be emailed
			// again next run — the notifier dedupes, so no duplicate is sent.
			metrics.RepoNotifyTotal.WithLabelValues("tag_error").Inc()
			zap.L().Error("failed to advance last_seen_tag after delivery",
				zap.Uint("subscription_id", sub.ID), zap.String("email", sub.Email), zap.Error(err))
			failed++
			continue
		}

		metrics.RepoNotifyTotal.WithLabelValues("success").Inc()
	}

	if failed > 0 {
		return fmt.Errorf("failed to notify %d of %d subscriber(s)", failed, len(subs))
	}

	return nil
}

func (s *Service) CheckAllReposTagAndAlert(ctx context.Context) error {
	repos, err := s.repositoriesRepository.GetAll(ctx)
	if err != nil {
		zap.L().Error("failed to fetch all repositories", zap.Error(err))
		return err
	}

	var errs []error
	for _, repo := range repos {
		err = s.CheckRepoTagAndAlert(ctx, repo)
		if err != nil {
			metrics.RepoCheckTotal.WithLabelValues("error").Inc()
			zap.L().Error("failed to update repository", zap.Error(err))
			errs = append(errs, err)
			continue
		}
		metrics.RepoCheckTotal.WithLabelValues("success").Inc()
	}
	if len(errs) > 0 {
		zap.L().Error("errors occurred on repositories update", zap.Int("errors_count", len(errs)))
		// todo: consolidate and return errors
	}

	return nil
}
