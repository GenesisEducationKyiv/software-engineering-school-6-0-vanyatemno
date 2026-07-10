// Package dispatch is the notifier's transport-agnostic delivery core: it turns
// one notification command into a single delivered email, with the same
// at-most-once guarantees regardless of how the command arrived.
//
// A Dispatcher records durable participant state in the deliveries table
// (SENDING → SENT/FAILED), claims a per-recipient Redis idempotency marker,
// renders the template and sends over SMTP, retrying transient failures. It is
// shared by two callers: the Kafka worker's saga path (which wraps Send with a
// saga reply and dead-letter) and the synchronous gRPC server (which returns the
// outcome to the API). Extracting it keeps a single, tested definition of "send
// this email exactly once" behind both transports.
package dispatch

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

	"go.uber.org/zap"
)

// retryBackoff is the pause between retry attempts. It is a var (not a const) so
// tests can shrink it.
var retryBackoff = 500 * time.Millisecond

// DeliveryStore records durable per-recipient dispatch state (SENDING → SENT/
// FAILED), the at-most-once guard that outlives the Redis marker's TTL.
type DeliveryStore interface {
	Claim(ctx context.Context, d *delivery.Delivery) (delivery.State, error)
	SetState(ctx context.Context, id int64, state delivery.State, lastErr string) error
}

// Command is one notification to deliver to a single recipient. SagaID is empty
// for non-saga (e.g. repository-update) sends and is recorded on the delivery row
// purely for traceability.
type Command struct {
	SagaID         string
	Template       contract.TemplateName
	Recipient      string
	Payload        json.RawMessage
	IdempotencyKey string
}

// Dispatcher delivers a single email at most once. maxRetries bounds transient
// retries within one Send (in addition to any redelivery/retry the caller does).
type Dispatcher struct {
	deliveries DeliveryStore
	dedup      dedup.Deduper
	templates  templates.TemplateService
	mailer     mailer.Mailer
	maxRetries int
}

func New(
	deliveries DeliveryStore,
	deduper dedup.Deduper,
	templates templates.TemplateService,
	mailer mailer.Mailer,
	maxRetries int,
) *Dispatcher {
	return &Dispatcher{
		deliveries: deliveries,
		dedup:      deduper,
		templates:  templates,
		mailer:     mailer,
		maxRetries: maxRetries,
	}
}

// Send delivers cmd to its recipient exactly once and records the outcome:
//   - claims (or finds) the durable delivery row; a row already SENT is an
//     idempotent success (nil, no re-send);
//   - renders + sends, retrying transient failures up to maxRetries;
//   - transitions the row to SENT on success or FAILED on error.
//
// A nil return means the email was delivered or had already been delivered — the
// caller may treat the notification as done. A non-nil return is a genuine
// failure the caller must surface (fail the saga / report to the API); errors are
// classified retryable vs terminal via IsRetryable. If ctx is cancelled mid-send
// the context error is returned and no terminal state is recorded, so a later
// attempt can safely resume.
func (d *Dispatcher) Send(ctx context.Context, cmd Command) error {
	rec := &delivery.Delivery{
		SagaID:         cmd.SagaID,
		IdempotencyKey: cmd.IdempotencyKey,
		Recipient:      cmd.Recipient,
		Template:       cmd.Template,
	}
	state, err := d.deliveries.Claim(ctx, rec)
	if err != nil {
		// Cannot record participant state → cannot guarantee at-most-once. Treat
		// as transient so the caller can retry.
		return Retryable(fmt.Errorf("delivery store: %w", err))
	}
	if state == delivery.StateSent {
		// A prior delivery already succeeded (even if the Redis marker expired).
		return nil
	}

	perr := d.deliverWithRetry(ctx, cmd)
	if ctx.Err() != nil {
		// Shutting down: leave the row in SENDING for a safe later attempt.
		return ctx.Err()
	}

	if perr == nil {
		if err := d.deliveries.SetState(ctx, rec.ID, delivery.StateSent, ""); err != nil {
			zap.L().Error("failed to mark delivery sent",
				zap.String("recipient", cmd.Recipient), zap.Error(err))
		}
		return nil
	}

	if err := d.deliveries.SetState(ctx, rec.ID, delivery.StateFailed, perr.Error()); err != nil {
		zap.L().Error("failed to mark delivery failed",
			zap.String("recipient", cmd.Recipient), zap.Error(err))
	}
	return perr
}

// deliverWithRetry runs deliver, retrying transient failures up to maxRetries.
// Retries are dedup-safe: a recipient already sent on an earlier attempt stays
// claimed and is skipped.
func (d *Dispatcher) deliverWithRetry(ctx context.Context, cmd Command) error {
	var err error
	for attempt := 0; attempt <= d.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryBackoff):
			}
			zap.L().Warn("retrying notification",
				zap.String("recipient", cmd.Recipient), zap.Int("attempt", attempt))
		}
		err = d.deliver(ctx, cmd)
		if err == nil {
			return nil
		}
		if !IsRetryable(err) {
			return err
		}
	}
	return err
}

// deliver renders the template and sends one email, deduping so the recipient is
// emailed at most once. A non-retryable error (unknown template, render failure)
// aborts immediately; a transient error (dedup store, SMTP) is returned as
// retryable.
func (d *Dispatcher) deliver(ctx context.Context, cmd Command) error {
	data, err := DecodePayload(cmd.Template, cmd.Payload)
	if err != nil {
		return fmt.Errorf("decode payload (template %q): %w", cmd.Template, err)
	}

	rendered, err := d.templates.RenderTemplate(cmd.Template, data)
	if err != nil {
		return fmt.Errorf("render template %q: %w", cmd.Template, err)
	}

	key := DedupKey(cmd.IdempotencyKey, cmd.Recipient)
	claimed, err := d.dedup.Claim(ctx, key)
	if err != nil {
		// Can't verify idempotency → don't risk a duplicate; retry later.
		return Retryable(fmt.Errorf("dedup claim: %w", err))
	}
	if !claimed {
		// Already sent to this recipient (redelivery / concurrent attempt).
		zap.L().Info("skipping already-sent email", zap.String("recipient", cmd.Recipient))
		return nil
	}

	if err := d.mailer.Send(&mailer.Message{
		To:      []string{cmd.Recipient},
		Subject: rendered.Subject,
		Body:    rendered.Body,
	}); err != nil {
		// Release the claim so a retry/redelivery can re-attempt this recipient.
		if relErr := d.dedup.Release(ctx, key); relErr != nil {
			zap.L().Error("failed to release dedup claim",
				zap.String("recipient", cmd.Recipient), zap.Error(relErr))
		}
		return Retryable(fmt.Errorf("send email: %w", err))
	}

	zap.L().Info("notification sent",
		zap.String("template", cmd.Template), zap.String("recipient", cmd.Recipient))
	return nil
}

// DecodePayload turns the raw JSON payload into the concrete struct the email
// template expects, based on the template name.
func DecodePayload(name contract.TemplateName, raw json.RawMessage) (any, error) {
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

// DedupKey builds the per-recipient idempotency key for a notification.
func DedupKey(idempotencyKey, recipient string) string {
	return fmt.Sprintf("notif:dedup:%s:%s", idempotencyKey, recipient)
}

// retryableError marks a processing failure worth retrying (a transient send or
// dedup-store error) as opposed to a permanent one (bad payload, bad template).
type retryableError struct{ err error }

func (e retryableError) Error() string { return e.err.Error() }
func (e retryableError) Unwrap() error { return e.err }

// Retryable wraps err to mark it as a transient failure worth retrying.
func Retryable(err error) error { return retryableError{err: err} }

// IsRetryable reports whether err was marked retryable via Retryable.
func IsRetryable(err error) bool {
	var r retryableError
	return errors.As(err, &r)
}
