package subscription

import (
	"se-school/internal/models/dto"

	"go.uber.org/zap"
)

func (s *Service) ListByEmail(req *dto.GetSubscriptionsRequest) ([]dto.SubscriptionResponse, error) {
	subscriptions, err := s.subscriptionsRepository.GetByEmail(req.Email)
	if err != nil {
		zap.L().Error("failed to fetch user's subscriptions", zap.Error(err))
		return nil, err
	}

	response := make([]dto.SubscriptionResponse, 0, len(subscriptions))
	for _, sub := range subscriptions {
		repo := ""
		if sub.Repository != nil {
			repo = sub.Repository.Owner + "/" + sub.Repository.Name
		}
		response = append(response, dto.SubscriptionResponse{
			Email:       sub.Email,
			Repo:        repo,
			Confirmed:   sub.IsConfirmed,
			LastSeenTag: sub.LastSeenTag,
		})
	}

	return response, nil
}
