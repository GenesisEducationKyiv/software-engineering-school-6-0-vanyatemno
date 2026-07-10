package models

import "time"

type SagaState = string

const (
	// SagaStateAwaitingNotification: T1 committed (subscription + codes + saga +
	// outbox), waiting for the notifier's dispatch reply.
	SagaStateAwaitingNotification SagaState = "AWAITING_NOTIFICATION"
	// SagaStateCompleted: the confirmation email was dispatched. The subscription
	// is durable and awaits the user's confirmation click.
	SagaStateCompleted SagaState = "COMPLETED"
	// SagaStateCompensating: dispatch failed (or deadline exceeded); compensation
	// (deleting the subscription + codes) is in progress.
	SagaStateCompensating SagaState = "COMPENSATING"
	// SagaStateCompensated: compensation finished — the subscription no longer
	// exists, so no user is left with a dangling, unconfirmable subscription.
	SagaStateCompensated SagaState = "COMPENSATED"
	// SagaStateFailed: compensation itself kept failing past the attempt cap — a
	// dead state for alerting/manual intervention.
	SagaStateFailed SagaState = "FAILED"
)

const SagaTypeSubscribeConfirm = "subscribe_confirm"

type SagaInstance struct {
	ID             string
	Type           string
	State          SagaState
	SubscriptionID uint
	Email          string
	IdempotencyKey string
	Attempts       int
	LastError      string
	DeadlineAt     time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
