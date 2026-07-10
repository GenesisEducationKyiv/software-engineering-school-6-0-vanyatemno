// Package worker consumes notification jobs from Kafka, renders the email
// template and sends it over SMTP. Delivery is at-least-once with a per-recipient
// idempotency guard (at-most-once emails); saga messages (with a SagaID) also
// record a durable delivery outcome and reply to the orchestrator.
package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ghnotify/contract"
	"ghnotify/notifier/internal/dedup"
	"ghnotify/notifier/internal/mailer"
	"ghnotify/notifier/internal/repositories/delivery"
	"ghnotify/notifier/internal/templates"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Reader is the subset of *kafka.Reader the worker uses, as an interface so
// tests can run without a real broker.
type Reader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
}

type DLQProducer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type ReplyProducer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type DeliveryStore interface {
	Claim(ctx context.Context, d *delivery.Delivery) (delivery.State, error)
	SetState(ctx context.Context, id int64, state delivery.State, lastErr string) error
}

// retryBackoff is a var (not a const) so tests can shrink it.
var retryBackoff = 500 * time.Millisecond

// retryableError marks a transient failure (send or dedup-store error) as
// retryable, versus a permanent one (bad payload or template).
type retryableError struct{ err error }

func (e retryableError) Error() string { return e.err.Error() }
func (e retryableError) Unwrap() error { return e.err }

func retryable(err error) error { return retryableError{err: err} }

func isRetryable(err error) bool {
	var r retryableError
	return errors.As(err, &r)
}

type Worker struct {
	reader     Reader
	dlq        DLQProducer
	replies    ReplyProducer
	dedup      dedup.Deduper
	deliveries DeliveryStore
	templates  templates.TemplateService
	mailer     mailer.Mailer
	maxRetries int
}

func New(
	reader Reader,
	dlq DLQProducer,
	replies ReplyProducer,
	deduper dedup.Deduper,
	deliveries DeliveryStore,
	templates templates.TemplateService,
	mailer mailer.Mailer,
	maxRetries int,
) *Worker {
	return &Worker{
		reader:     reader,
		dlq:        dlq,
		replies:    replies,
		dedup:      deduper,
		deliveries: deliveries,
		templates:  templates,
		mailer:     mailer,
		maxRetries: maxRetries,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	zap.L().Info("consuming notifications topic")
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("worker shutting down", zap.Error(ctx.Err()))
			return nil
		default:
		}

		msg, err := w.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				continue // shutting down — loop back so the select returns cleanly
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		w.handle(ctx, msg)
	}
}

// handle processes one Kafka message: fire-and-forget notifications keep their
// original send-and-dead-letter behavior; saga messages additionally record the
// delivery outcome and reply to the orchestrator. The offset is committed in
// every outcome (except a mid-shutdown abort, where redelivery + idempotency
// make reprocessing safe).
func (w *Worker) handle(ctx context.Context, msg kafka.Message) {
	var m contract.Message
	if err := json.Unmarshal(msg.Value, &m); err != nil {
		// Poison: we cannot decode the envelope, so we cannot identify a saga to
		// reply to. Dead-letter and advance; a saga (if any) is caught by the
		// orchestrator's deadline sweeper.
		if ctx.Err() == nil {
			w.deadLetter(ctx, msg, fmt.Errorf("unmarshal message: %w", err))
		}
		w.commit(ctx, msg)
		return
	}

	if m.SagaID == "" {
		// Fire-and-forget path (e.g. cron release alerts) — unchanged behavior.
		if perr := w.processWithRetry(ctx, m); perr != nil && ctx.Err() == nil {
			w.deadLetter(ctx, msg, perr)
		}
		w.commit(ctx, msg)
		return
	}

	w.handleSaga(ctx, msg, m)
}

func (w *Worker) handleSaga(ctx context.Context, msg kafka.Message, m contract.Message) {
	recipient := ""
	if len(m.Receivers) > 0 {
		recipient = m.Receivers[0]
	}
	if len(m.Receivers) != 1 {
		zap.L().Warn("saga message expected exactly one recipient",
			zap.String("saga_id", m.SagaID), zap.Int("receivers", len(m.Receivers)))
	}

	d := &delivery.Delivery{
		SagaID:         m.SagaID,
		IdempotencyKey: m.IdempotencyKey,
		Recipient:      recipient,
		Template:       m.Template,
	}
	state, err := w.deliveries.Claim(ctx, d)
	if err != nil {
		// Cannot record participant state → cannot guarantee delivery. Fail the
		// saga so it compensates deterministically, and dead-letter for inspection.
		if ctx.Err() == nil {
			zap.L().Error("failed to claim delivery", zap.String("saga_id", m.SagaID), zap.Error(err))
			w.reply(ctx, m, recipient, fmt.Errorf("delivery store: %w", err))
			w.deadLetter(ctx, msg, err)
		}
		w.commit(ctx, msg)
		return
	}

	// Durable at-most-once: a prior delivery already succeeded (even if the Redis
	// marker has since expired). Re-affirm the reply and advance.
	if state == delivery.StateSent {
		zap.L().Info("delivery already sent; re-affirming reply", zap.String("saga_id", m.SagaID))
		w.reply(ctx, m, recipient, nil)
		w.commit(ctx, msg)
		return
	}

	perr := w.processWithRetry(ctx, m)
	if ctx.Err() != nil {
		return // shutting down — leave uncommitted for safe redelivery
	}

	if perr == nil {
		if err := w.deliveries.SetState(ctx, d.ID, delivery.StateSent, ""); err != nil {
			zap.L().Error("failed to mark delivery sent", zap.String("saga_id", m.SagaID), zap.Error(err))
		}
		w.reply(ctx, m, recipient, nil)
	} else {
		if err := w.deliveries.SetState(ctx, d.ID, delivery.StateFailed, perr.Error()); err != nil {
			zap.L().Error("failed to mark delivery failed", zap.String("saga_id", m.SagaID), zap.Error(err))
		}
		w.reply(ctx, m, recipient, perr)
		w.deadLetter(ctx, msg, perr)
	}
	w.commit(ctx, msg)
}

// Retries are dedup-safe: recipients sent on an earlier attempt stay claimed and
// are skipped, so only the failed recipients are re-attempted.
func (w *Worker) processWithRetry(ctx context.Context, m contract.Message) error {
	var err error
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryBackoff):
			}
			zap.L().Warn("retrying notification", zap.Int("attempt", attempt))
		}
		err = w.process(ctx, m)
		if err == nil {
			return nil
		}
		if !isRetryable(err) {
			return err
		}
	}
	return err
}

// process renders one message and sends an email to each recipient, deduping so
// each recipient is emailed at most once. A non-retryable error (unknown
// template, render failure) aborts immediately; a retryable error is returned
// when one or more sends fail so the message can be retried.
func (w *Worker) process(ctx context.Context, m contract.Message) error {
	data, err := decodePayload(m.Template, m.Payload)
	if err != nil {
		return fmt.Errorf("decode payload (template %q): %w", m.Template, err)
	}

	rendered, err := w.templates.RenderTemplate(m.Template, data)
	if err != nil {
		return fmt.Errorf("render template %q: %w", m.Template, err)
	}

	var failed int
	for _, recipient := range m.Receivers {
		key := dedupKey(m.IdempotencyKey, recipient)

		claimed, err := w.dedup.Claim(ctx, key)
		if err != nil {
			// Can't verify idempotency → don't risk a duplicate; retry later.
			zap.L().Error("dedup claim failed",
				zap.String("recipient", recipient), zap.Error(err))
			failed++
			continue
		}
		if !claimed {
			zap.L().Info("skipping already-sent email", zap.String("recipient", recipient))
			continue
		}

		if err := w.mailer.Send(&mailer.Message{
			To:      []string{recipient},
			Subject: rendered.Subject,
			Body:    rendered.Body,
		}); err != nil {
			// Release the claim so a retry/redelivery can re-attempt this recipient.
			if relErr := w.dedup.Release(ctx, key); relErr != nil {
				zap.L().Error("failed to release dedup claim",
					zap.String("recipient", recipient), zap.Error(relErr))
			}
			zap.L().Error("failed to send email",
				zap.String("recipient", recipient), zap.Error(err))
			failed++
			continue
		}
	}

	if failed > 0 {
		return retryable(fmt.Errorf("failed to send to %d recipient(s)", failed))
	}

	zap.L().Info("notification sent",
		zap.String("template", m.Template),
		zap.Int("receivers", len(m.Receivers)))
	return nil
}

// Saga replies are keyed by SagaID for per-saga (per-partition) ordering.
func (w *Worker) reply(ctx context.Context, m contract.Message, recipient string, cause error) {
	status := contract.ReplyDispatched
	reason := ""
	if cause != nil {
		status = contract.ReplyFailed
		reason = cause.Error()
	}

	body, err := json.Marshal(contract.Reply{
		SagaID:         m.SagaID,
		IdempotencyKey: m.IdempotencyKey,
		Recipient:      recipient,
		Status:         status,
		Reason:         reason,
	})
	if err != nil {
		zap.L().Error("failed to marshal reply", zap.String("saga_id", m.SagaID), zap.Error(err))
		return
	}

	if err := w.replies.WriteMessages(ctx, kafka.Message{Key: []byte(m.SagaID), Value: body}); err != nil {
		zap.L().Error("failed to publish reply", zap.String("saga_id", m.SagaID), zap.Error(err))
		return
	}
	zap.L().Info("published saga reply",
		zap.String("saga_id", m.SagaID), zap.String("status", status))
}

func (w *Worker) deadLetter(ctx context.Context, msg kafka.Message, cause error) {
	zap.L().Error("dead-lettering notification", zap.Error(cause))
	if err := w.dlq.WriteMessages(ctx, kafka.Message{Key: msg.Key, Value: msg.Value}); err != nil {
		zap.L().Error("failed to write to dead-letter topic", zap.Error(err))
	}
}

// Called in every terminal outcome so the consumer never loops on a poison message.
func (w *Worker) commit(ctx context.Context, msg kafka.Message) {
	if err := w.reader.CommitMessages(ctx, msg); err != nil && ctx.Err() == nil {
		zap.L().Error("failed to commit offset", zap.Error(err))
	}
}

func dedupKey(idempotencyKey, recipient string) string {
	return fmt.Sprintf("notif:dedup:%s:%s", idempotencyKey, recipient)
}

func decodePayload(name contract.TemplateName, raw json.RawMessage) (any, error) {
	switch name {
	case contract.Confirmation:
		var p contract.ConfirmEmailPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		return p, nil
	case contract.RepositoryUpdated:
		var p contract.RepositoryUpdateEmailPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, fmt.Errorf("unknown template %q", name)
	}
}
