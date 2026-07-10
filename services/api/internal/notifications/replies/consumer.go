// Package replies consumes saga replies from Kafka and drives the orchestrator.
// It is the return path of the distributed transaction: the notifications
// service reports whether it dispatched the confirmation email, and this
// consumer hands each reply to the orchestrator to complete or compensate the
// saga.
package replies

import (
	"context"
	"encoding/json"
	"fmt"

	"ghnotify/contract"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Reader is the subset of *kafka.Reader the consumer needs (mirrors the
// notifier worker's Reader so it can be driven without a real broker in tests).
type Reader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
}

// Handler applies a reply to the saga state machine (satisfied by the
// subscription orchestrator's HandleReply).
type Handler interface {
	HandleReply(ctx context.Context, reply contract.Reply) error
}

type Consumer struct {
	reader  Reader
	handler Handler
}

func New(reader Reader, handler Handler) *Consumer {
	return &Consumer{reader: reader, handler: handler}
}

// Run consumes replies until ctx is canceled. The offset is committed on every
// outcome (handled, poison, or handler error) so the consumer always advances
// and never loops on a message forever — a saga left un-advanced by a handler
// error is caught by the orchestrator's deadline sweeper. HandleReply is
// idempotent, so this at-least-once consumption is safe.
func (c *Consumer) Run(ctx context.Context) error {
	zap.L().Info("saga reply consumer started")
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("reply consumer stopping", zap.Error(ctx.Err()))
			return nil
		default:
		}

		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				continue
			}
			return fmt.Errorf("fetch reply: %w", err)
		}

		var reply contract.Reply
		if err := json.Unmarshal(msg.Value, &reply); err != nil {
			zap.L().Error("failed to decode reply, skipping", zap.Error(err))
		} else if err := c.handler.HandleReply(ctx, reply); err != nil && ctx.Err() == nil {
			zap.L().Error("failed to handle reply",
				zap.String("saga_id", reply.SagaID), zap.Error(err))
		}

		if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil && ctx.Err() == nil {
			zap.L().Error("failed to commit reply offset", zap.Error(commitErr))
		}
	}
}
