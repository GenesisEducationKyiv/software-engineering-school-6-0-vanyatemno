package subscription

import (
	"se-school/internal/models"
	"se-school/internal/models/dto"

	"go.uber.org/zap"
)

func (s *Service) Confirm(req *dto.ConfirmSubscriptionRequest) error {
	code, err := s.codesRepository.Get(req.Token)
	if err != nil {
		zap.L().Error("failed to find code", zap.Error(err))
		return err
	}
	subscription, err := s.subscriptionsRepository.GetByCode(code.ID, models.CodeTypeConfirm)
	if err != nil {
		zap.L().Error("failed to find subscription", zap.Error(err))
		return err
	}

	subscription.IsConfirmed = true
	err = s.subscriptionsRepository.Save(subscription)
	if err != nil {
		zap.L().Error("failed to save subscription", zap.Error(err))
		return err
	}

	err = s.codesRepository.Delete(code.ID)
	if err != nil {
		zap.L().Error("failed to delete code", zap.Error(err))
	}

	return nil
}
