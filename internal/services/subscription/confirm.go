package subscription

import (
	"context"
	"se-school/internal/models"
	"se-school/internal/models/dto"

	"go.uber.org/zap"
)

func (s *Service) Confirm(ctx context.Context, req *dto.ConfirmSubscriptionRequest) error {
	code, err := s.codesRepository.Get(ctx, req.Token)
	if err != nil {
		zap.L().Error("failed to find code", zap.Error(err))
		return err
	}
	subscription, err := s.subscriptionsRepository.GetByCode(ctx, code.ID, models.CodeTypeConfirm)
	if err != nil {
		zap.L().Error("failed to find subscription", zap.Error(err))
		return err
	}

	subscription.IsConfirmed = true
	err = s.subscriptionsRepository.Save(ctx, subscription)
	if err != nil {
		zap.L().Error("failed to save subscription", zap.Error(err))
		return err
	}

	err = s.codesRepository.Delete(ctx, code.ID)
	if err != nil {
		zap.L().Error("failed to delete code", zap.Error(err))
	}

	return nil
}
