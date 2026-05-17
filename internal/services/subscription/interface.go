package subscription

import (
	"context"
	"se-school/internal/models"
	"se-school/internal/models/dto"
)

type SubscriptionsService interface {
	ListByEmail(*dto.GetSubscriptionsRequest) ([]*models.Subscription, error)
	Create(context.Context, *dto.CreateSubscriptionRequest) error
	Confirm(*dto.ConfirmSubscriptionRequest) error
	Unsubscribe(*dto.UnsubscribeRequest) error
}
