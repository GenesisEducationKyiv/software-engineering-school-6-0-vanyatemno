// Package relay drains the transactional outbox to Kafka in a background loop,
// decoupling the DB commit from the publish so command emission is at-least-once,
// not a dual-write.
package relay

import (
	"context"
	"time"

	"se-school/internal/models"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Producer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type Store interface {
	GetUnpublished(ctx context.Context, limit int) ([]*models.OutboxMessage, error)
	MarkPublished(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id, cause string) error
}

type Relay struct {
	store    Store
	producer Producer
	interval time.Duration
	batch    int
}

func New(store Store, producer Producer, interval time.Duration, batch int) *Relay {
	return &Relay{store: store, producer: producer, interval: interval, batch: batch}
}

func (r *Relay) Run(ctx context.Context) {
	zap.L().Info("outbox relay started", zap.Duration("interval", r.interval))
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("outbox relay stopping", zap.Error(ctx.Err()))
			return
		case <-ticker.C:
			r.drain(ctx)
		}
	}
}

// drain publishes each unpublished message. A publish failure is recorded and
// left unpublished so the next tick retries it; the consumer's idempotency makes
// a re-publish (e.g. after a crash between publish and MarkPublished) harmless.
func (r *Relay) drain(ctx context.Context) {
	msgs, err := r.store.GetUnpublished(ctx, r.batch)
	if err != nil {
		if ctx.Err() == nil {
			zap.L().Error("relay: failed to load outbox", zap.Error(err))
		}
		return
	}

	for _, m := range msgs {
		if err := r.producer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(m.KafkaKey),
			Value: m.Payload,
		}); err != nil {
			if ctx.Err() != nil {
				return
			}
			zap.L().Error("relay: failed to publish outbox message",
				zap.String("id", m.ID), zap.String("saga_id", m.SagaID), zap.Error(err))
			_ = r.store.MarkFailed(ctx, m.ID, err.Error())
			continue
		}
		if err := r.store.MarkPublished(ctx, m.ID); err != nil {
			zap.L().Error("relay: published but failed to mark outbox row",
				zap.String("id", m.ID), zap.Error(err))
		}
	}
}
