package subscription

import (
	"context"
	"se-school/internal/models"
	"se-school/internal/models/dto"

	"go.uber.org/zap"
)

func (s *Service) Unsubscribe(ctx context.Context, req *dto.UnsubscribeRequest) error {
	code, err := s.codesRepository.Get(ctx, req.Token)
	if err != nil {
		zap.L().Error("failed to find unsub code", zap.Error(err))
		return err
	}
	subscription, err := s.subscriptionsRepository.GetByCode(ctx, code.ID, models.CodeTypeUnsubscribe)
	if err != nil {
		zap.L().Error("failed to find subscription", zap.Error(err))
		return err
	}
	err = s.subscriptionsRepository.Delete(ctx, subscription)
	if err != nil {
		zap.L().Error("failed to delete subscription", zap.Error(err))
		return err
	}

	return nil
}
