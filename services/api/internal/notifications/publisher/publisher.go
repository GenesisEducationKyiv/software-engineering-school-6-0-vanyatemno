// Package publisher implements the API side of the notifications boundary. It
// satisfies the NotificationsService interface the domain services depend on,
// but instead of rendering and sending email in-process it marshals a
// contract.Message and writes it to the Kafka topic, where the notifications
// microservice consumes it and delivers the email.
package publisher

import (
	"context"
	"encoding/json"

	"ghnotify/contract"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Publisher struct {
	ctx    context.Context
	writer *kafka.Writer
}

// New returns a Publisher bound to the application context, so in-flight
// publishes are canceled on shutdown.
func New(ctx context.Context, writer *kafka.Writer) *Publisher {
	return &Publisher{ctx: ctx, writer: writer}
}

// SendEmail publishes a notification job. It stamps a deterministic idempotency
// key (and uses it as the Kafka message key) so the consumer can guarantee each
// email is sent at most once even if the message is redelivered or re-published.
// It returns an error only if the job could not be enqueued; actual delivery
// happens asynchronously in the notifications service.
func (p *Publisher) SendEmail(receivers []string, template contract.TemplateName, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		zap.L().Error("failed to marshal notification payload", zap.Error(err))
		return err
	}

	key := contract.IdempotencyKey(template, payload)
	body, err := json.Marshal(contract.Message{
		Template:       template,
		Receivers:      receivers,
		Payload:        payload,
		IdempotencyKey: key,
	})
	if err != nil {
		zap.L().Error("failed to marshal notification message", zap.Error(err))
		return err
	}

	if err := p.writer.WriteMessages(p.ctx, kafka.Message{
		Key:   []byte(key),
		Value: body,
	}); err != nil {
		zap.L().Error("failed to publish notification",
			zap.String("template", template), zap.Error(err))
		return err
	}

	zap.L().Debug("published notification",
		zap.String("template", template),
		zap.Int("receivers", len(receivers)),
		zap.String("idempotency_key", key))
	return nil
}
