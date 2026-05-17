package subscription

import (
	"context"
	"se-school/internal/models"
	"se-school/internal/models/dto"

	"go.uber.org/zap"
)

func (s *Service) ListByEmail(ctx context.Context, req *dto.GetSubscriptionsRequest) ([]*models.Subscription, error) {
	subscriptions, err := s.subscriptionsRepository.GetByEmail(ctx, req.Email)
	if err != nil {
		zap.L().Error("failed to fetch user's subscriptions", zap.Error(err))
		return nil, err
	}

	return subscriptions, nil
}
