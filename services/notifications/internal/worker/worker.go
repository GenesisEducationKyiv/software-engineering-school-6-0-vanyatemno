// Package worker consumes notification jobs from Kafka, renders the requested
// email template and sends it over SMTP. It is the asynchronous counterpart of
// the API's publisher: the API publishes a contract.Message, this worker turns
// it into an actual email.
//
// Delivery is at-least-once (durable log + manual offset commit), so a message
// can be redelivered — e.g. if the worker crashes after sending but before
// committing. To keep every email at-most-once, the worker claims a per-recipient
// idempotency marker before sending and skips any recipient already claimed.
// Messages that fail terminally (bad payload, unknown template) or exhaust their
// retries are routed to a dead-letter topic. The offset is committed in every
// outcome so the consumer always advances and never loops on a poison message.
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
	"ghnotify/notifier/internal/templates"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Reader is the subset of *kafka.Reader the worker needs, defined as an
// interface so tests can drive the worker without a real broker.
type Reader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
}

// DLQProducer writes messages that cannot be processed to the dead-letter topic.
type DLQProducer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

// retryBackoff is the pause between retry attempts. It is a var (not a const) so
// tests can shrink it.
var retryBackoff = 500 * time.Millisecond

// retryableError marks a processing failure worth retrying (a transient send or
// dedup-store error) as opposed to a permanent one (bad payload, bad template).
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
	dedup      dedup.Deduper
	templates  templates.TemplateService
	mailer     mailer.Mailer
	maxRetries int
}

func New(
	reader Reader,
	dlq DLQProducer,
	deduper dedup.Deduper,
	templates templates.TemplateService,
	mailer mailer.Mailer,
	maxRetries int,
) *Worker {
	return &Worker{
		reader:     reader,
		dlq:        dlq,
		dedup:      deduper,
		templates:  templates,
		mailer:     mailer,
		maxRetries: maxRetries,
	}
}

// Run consumes messages until ctx is cancelled. Each message is processed (with
// bounded retries for transient failures); on terminal failure it is
// dead-lettered. The offset is committed in every outcome so the consumer
// advances and never reprocesses a poison message forever.
func (w *Worker) Run(ctx context.Context) error {
	zap.L().Info("consuming notifications topic")
	for {
		msg, err := w.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				zap.L().Info("worker shutting down", zap.Error(ctx.Err()))
				return nil
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		if perr := w.processWithRetry(ctx, msg.Value); perr != nil {
			if ctx.Err() != nil {
				// Shutting down mid-message: don't dead-letter or commit; the
				// message will be redelivered on restart (dedup makes that safe).
				return nil
			}
			w.deadLetter(ctx, msg, perr)
		}

		if err := w.reader.CommitMessages(ctx, msg); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			zap.L().Error("failed to commit offset", zap.Error(err))
		}
	}
}

// processWithRetry runs process, retrying transient failures up to maxRetries.
// Retries are dedup-safe: recipients sent on an earlier attempt stay claimed and
// are skipped, so only the recipients that failed are re-attempted.
func (w *Worker) processWithRetry(ctx context.Context, value []byte) error {
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
		err = w.process(ctx, value)
		if err == nil {
			return nil
		}
		if !isRetryable(err) {
			return err
		}
	}
	return err
}

// process decodes one message and sends an email to each recipient, deduping so
// each recipient is emailed at most once. A non-retryable error (bad JSON,
// unknown template, render failure) aborts immediately; a retryable error is
// returned when one or more sends fail so the message can be retried.
func (w *Worker) process(ctx context.Context, value []byte) error {
	var m contract.Message
	if err := json.Unmarshal(value, &m); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

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

// deadLetter publishes a failed message to the DLQ topic, preserving its key.
func (w *Worker) deadLetter(ctx context.Context, msg kafka.Message, cause error) {
	zap.L().Error("dead-lettering notification", zap.Error(cause))
	if err := w.dlq.WriteMessages(ctx, kafka.Message{Key: msg.Key, Value: msg.Value}); err != nil {
		zap.L().Error("failed to write to dead-letter topic", zap.Error(err))
	}
}

// dedupKey builds the per-recipient idempotency key for a message.
func dedupKey(idempotencyKey, recipient string) string {
	return fmt.Sprintf("notif:dedup:%s:%s", idempotencyKey, recipient)
}

// decodePayload turns the raw JSON payload into the concrete struct the email
// template expects, based on the template name.
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
