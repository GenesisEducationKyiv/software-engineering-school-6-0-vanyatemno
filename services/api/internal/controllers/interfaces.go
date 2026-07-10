package controllers

import (
	"context"

	"se-school/internal/models"
	"se-school/internal/models/dto"
)

type SubscriptionsService interface {
	ListByEmail(context.Context, *dto.GetSubscriptionsRequest) ([]dto.SubscriptionResponse, error)
	// Create starts the orchestrated confirmation saga and returns its id.
	Create(context.Context, *dto.CreateSubscriptionRequest) (string, error)
	// Status returns the current saga instance for status polling.
	Status(context.Context, string) (*models.SagaInstance, error)
	Confirm(context.Context, *dto.ConfirmSubscriptionRequest) error
	Unsubscribe(context.Context, *dto.UnsubscribeRequest) error
}
