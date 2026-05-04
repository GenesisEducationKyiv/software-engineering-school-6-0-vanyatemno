package subscription

import (
	"se-school/internal/models"
	"se-school/internal/models/dto"

	"go.uber.org/zap"
)

func (s *Service) Unsubscribe(req *dto.UnsubscribeRequest) error {
	code, err := s.codesRepository.Get(req.Token)
	if err != nil {
		zap.L().Error("failed to find unsub code", zap.Error(err))
		return err
	}
	subscription, err := s.subscriptionsRepository.GetByCode(code.ID, models.CodeTypeUnsubscribe)
	if err != nil {
		zap.L().Error("failed to find subscription", zap.Error(err))
		return err
	}
	err = s.subscriptionsRepository.Delete(subscription)
	if err != nil {
		zap.L().Error("failed to delete subscription", zap.Error(err))
		return err
	}

	return nil
}
