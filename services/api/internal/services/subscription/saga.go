package subscription

import (
	"context"
	"errors"
	"time"

	"ghnotify/contract"

	"se-school/internal/models"

	"go.uber.org/zap"
)

// HandleReply advances the saga based on the notifier's dispatch reply. It is
// idempotent: a reply for a saga already in a terminal state (or an unknown
// saga) is ignored, so duplicate or redelivered replies are safe.
func (s *Service) HandleReply(ctx context.Context, reply contract.Reply) error {
	saga, err := s.sagaRepository.GetByID(ctx, reply.SagaID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			zap.L().Warn("reply for unknown saga", zap.String("saga_id", reply.SagaID))
			return nil
		}
		return err
	}

	if isTerminal(saga.State) {
		zap.L().Info("ignoring reply for terminal saga",
			zap.String("saga_id", saga.ID), zap.String("state", saga.State))
		return nil
	}

	switch reply.Status {
	case contract.ReplyDispatched:
		zap.L().Info("saga completed: confirmation email dispatched", zap.String("saga_id", saga.ID))
		return s.sagaRepository.UpdateState(ctx, saga.ID, models.SagaStateCompleted, "")
	case contract.ReplyFailed:
		return s.compensate(ctx, saga, reply.Reason)
	default:
		zap.L().Warn("unknown reply status",
			zap.String("saga_id", saga.ID), zap.String("status", reply.Status))
		return nil
	}
}

// compensate rolls back T1: it soft-deletes the subscription and its codes
// (reusing the subscription repository's atomic Delete) and marks the saga
// COMPENSATED, so no user is left with a dangling, unconfirmable subscription.
// The shared repositories row is intentionally left intact.
func (s *Service) compensate(ctx context.Context, saga *models.SagaInstance, reason string) error {
	if err := s.sagaRepository.UpdateState(ctx, saga.ID, models.SagaStateCompensating, reason); err != nil {
		return err
	}

	sub, err := s.subscriptionsRepository.GetByID(ctx, saga.SubscriptionID)
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		return err
	}
	if sub != nil {
		if err := s.subscriptionsRepository.Delete(ctx, sub); err != nil {
			zap.L().Error("compensation failed to delete subscription",
				zap.String("saga_id", saga.ID), zap.Error(err))
			return err
		}
	}

	zap.L().Info("saga compensated", zap.String("saga_id", saga.ID), zap.String("reason", reason))
	return s.sagaRepository.UpdateState(ctx, saga.ID, models.SagaStateCompensated, reason)
}

// Sweep compensates sagas stuck in AWAITING_NOTIFICATION past their deadline —
// the durability safety net for a reply that never arrived (notifier down or
// reply lost). It runs periodically from the API's background loop.
func (s *Service) Sweep(ctx context.Context, limit int) error {
	stuck, err := s.sagaRepository.GetStuck(ctx, models.SagaStateAwaitingNotification, time.Now(), limit)
	if err != nil {
		return err
	}
	for _, saga := range stuck {
		zap.L().Warn("compensating stuck saga past deadline", zap.String("saga_id", saga.ID))
		if err := s.compensate(ctx, saga, "deadline exceeded: no dispatch reply"); err != nil {
			zap.L().Error("failed to compensate stuck saga",
				zap.String("saga_id", saga.ID), zap.Error(err))
		}
	}
	return nil
}

func (s *Service) Status(ctx context.Context, sagaID string) (*models.SagaInstance, error) {
	return s.sagaRepository.GetByID(ctx, sagaID)
}

func isTerminal(state models.SagaState) bool {
	switch state {
	case models.SagaStateCompleted, models.SagaStateCompensated, models.SagaStateFailed:
		return true
	default:
		return false
	}
}
