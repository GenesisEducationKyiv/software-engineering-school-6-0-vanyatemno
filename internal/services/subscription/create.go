package subscription

import (
	"context"
	"errors"
	"fmt"
	"se-school/internal/models"
	"se-school/internal/models/dto"
	"se-school/internal/notifications/templates"
	"se-school/internal/repositories"
	"strings"

	"go.uber.org/zap"
)

func (s *Service) Create(
	ctx context.Context,
	req *dto.CreateSubscriptionRequest,
) error {
	sub, err := s.createNewSubscription(ctx, req)
	if err != nil {
		zap.L().Error("failed to init new subscription", zap.Error(err))
		return err
	}
	err = s.sendConfirmationCode(sub)
	if err != nil {
		zap.L().Error("failed to send confirmation code", zap.Error(err))
		deleteError := s.subscriptionsRepository.Delete(sub)
		if deleteError != nil {
			zap.L().Error("failed to rollback subscription create", zap.Error(deleteError))
			return deleteError
		}
		return err
	}

	return nil
}

func (s *Service) createNewSubscription(
	ctx context.Context,
	req *dto.CreateSubscriptionRequest,
) (*models.Subscription, error) {
	parsedRepoValues, err := parseRepoFields(req.Repo)
	if err != nil {
		zap.L().Error("failed to parse repo fields", zap.Error(err))
		return nil, err
	}

	repo, err := s.getOrCreateRepository(ctx, parsedRepoValues)
	if err != nil {
		return nil, err
	}

	unsubCode, err := s.codesRepository.Create(models.CodeTypeUnsubscribe)
	if err != nil {
		return nil, err
	}
	subCode, err := s.codesRepository.Create(models.CodeTypeConfirm)
	if err != nil {
		return nil, err
	}

	sub := &models.Subscription{
		RepositoryID:      repo.ID,
		UnsubscribeCodeID: unsubCode.ID,
		SubscribeCode:     subCode,
		Email:             req.Email,
		LastSeenTag:       repo.Version,
	}
	err = s.subscriptionsRepository.Create(sub)
	if err != nil {
		zap.L().Error("failed to create subscription", zap.Error(err))
		return nil, err
	}
	sub.SubscribeCode = subCode
	sub.UnsubscribeCode = unsubCode

	return sub, nil
}

func parseRepoFields(repo string) (*parsedRepoValue, error) {
	values := strings.Split(repo, "/")
	if len(values) != 2 {
		return nil, fmt.Errorf("failed to parse repo fields: %s", repo)
	}

	return &parsedRepoValue{
		Owner:          values[0],
		RepositoryName: values[1],
	}, nil
}

func (s *Service) getOrCreateRepository(ctx context.Context, values *parsedRepoValue) (*models.Repository, error) {
	repository, err := s.repositoriesRepository.Find(&models.Repository{
		Owner: values.Owner,
		Name:  values.RepositoryName,
	})
	if err != nil {
		if !errors.Is(err, repositories.ErrNotFound) {
			return nil, err
		}
		repository, err = s.createRepository(ctx, values)
		if err != nil {
			return nil, err
		}
	}

	return repository, nil
}

func (s *Service) createRepository(ctx context.Context, values *parsedRepoValue) (*models.Repository, error) {
	currentVersion, err := s.githubIntegration.GetRepositoryVersion(ctx, values.Owner, values.RepositoryName)
	if err != nil {
		return nil, err
	}
	repo := &models.Repository{
		Owner:   values.Owner,
		Name:    values.RepositoryName,
		Version: currentVersion,
	}
	err = s.repositoriesRepository.Create(repo)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func (s *Service) sendConfirmationCode(sub *models.Subscription) error {
	err := s.notificationService.SendEmail(
		[]string{sub.Email},
		templates.Confirmation,
		templates.BuildConfirmEmailPayload(s.cfg, sub.SubscribeCode.Code),
	)
	if err != nil {
		return err
	}

	return nil
}
