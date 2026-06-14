// Package worker consumes notification jobs from the Redis Pub/Sub channel,
// renders the requested email template and sends it over SMTP. It is the
// asynchronous counterpart of the API's publisher: the API publishes a
// contract.Message, this worker turns it into an actual email.
package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"ghnotify/contract"
	"ghnotify/notifier/internal/mailer"
	"ghnotify/notifier/internal/templates"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Worker struct {
	rdb       *redis.Client
	templates templates.TemplateService
	mailer    mailer.Mailer
}

func New(rdb *redis.Client, templates templates.TemplateService, mailer mailer.Mailer) *Worker {
	return &Worker{
		rdb:       rdb,
		templates: templates,
		mailer:    mailer,
	}
}

// Run subscribes to the notifications channel and processes messages until ctx
// is cancelled. It returns once the subscription is torn down.
func (w *Worker) Run(ctx context.Context) error {
	pubsub := w.rdb.Subscribe(ctx, contract.Channel)
	defer func() { _ = pubsub.Close() }()

	// Block until the subscription is actually established so we don't miss
	// messages published immediately after start-up.
	if _, err := pubsub.Receive(ctx); err != nil {
		return fmt.Errorf("subscribe to %s: %w", contract.Channel, err)
	}
	zap.L().Info("subscribed to notifications channel", zap.String("channel", contract.Channel))

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("worker shutting down", zap.Error(ctx.Err()))
			return nil
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			w.handle(msg.Payload)
		}
	}
}

// handle processes a single published message. Failures are logged and dropped:
// Pub/Sub is fire-and-forget, so there is no acknowledgement to retry against.
func (w *Worker) handle(raw string) {
	var m contract.Message
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		zap.L().Error("failed to unmarshal notification message", zap.Error(err))
		return
	}

	data, err := decodePayload(m.Template, m.Payload)
	if err != nil {
		zap.L().Error("failed to decode notification payload",
			zap.String("template", m.Template), zap.Error(err))
		return
	}

	rendered, err := w.templates.RenderTemplate(m.Template, data)
	if err != nil {
		zap.L().Error("failed to render template",
			zap.String("template", m.Template), zap.Error(err))
		return
	}

	if err := w.mailer.Send(&mailer.Message{
		To:      m.Receivers,
		Subject: rendered.Subject,
		Body:    rendered.Body,
	}); err != nil {
		zap.L().Error("failed to send email(s)",
			zap.String("template", m.Template),
			zap.Strings("receivers", m.Receivers),
			zap.Error(err))
		return
	}

	zap.L().Info("notification sent",
		zap.String("template", m.Template),
		zap.Int("receivers", len(m.Receivers)))
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
