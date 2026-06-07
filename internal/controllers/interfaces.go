package controllers

import (
	"context"
	"se-school/internal/models/dto"
)

type SubscriptionsService interface {
	ListByEmail(context.Context, *dto.GetSubscriptionsRequest) ([]dto.SubscriptionResponse, error)
	Create(context.Context, *dto.CreateSubscriptionRequest) error
	Confirm(context.Context, *dto.ConfirmSubscriptionRequest) error
	Unsubscribe(context.Context, *dto.UnsubscribeRequest) error
}
