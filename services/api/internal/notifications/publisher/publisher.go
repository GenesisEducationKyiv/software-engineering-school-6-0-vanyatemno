// Package publisher implements the API side of the notifications boundary. It
// satisfies the NotificationsService interface the domain services depend on,
// but instead of rendering and sending email in-process it marshals a
// contract.Message and publishes it to the Redis Pub/Sub channel, where the
// notifications microservice picks it up and delivers it.
package publisher

import (
	"context"
	"encoding/json"

	"ghnotify/contract"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Publisher struct {
	ctx context.Context
	rdb *redis.Client
}

// New returns a Publisher bound to the application context, so in-flight
// publishes are canceled on shutdown.
func New(ctx context.Context, rdb *redis.Client) *Publisher {
	return &Publisher{ctx: ctx, rdb: rdb}
}

// SendEmail publishes a notification job. It returns an error only if the job
// could not be enqueued; actual delivery happens asynchronously in the
// notifications service.
func (p *Publisher) SendEmail(receivers []string, template contract.TemplateName, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		zap.L().Error("failed to marshal notification payload", zap.Error(err))
		return err
	}

	body, err := json.Marshal(contract.Message{
		Template:  template,
		Receivers: receivers,
		Payload:   payload,
	})
	if err != nil {
		zap.L().Error("failed to marshal notification message", zap.Error(err))
		return err
	}

	if err := p.rdb.Publish(p.ctx, contract.Channel, body).Err(); err != nil {
		zap.L().Error("failed to publish notification",
			zap.String("template", template), zap.Error(err))
		return err
	}

	zap.L().Debug("published notification",
		zap.String("template", template), zap.Int("receivers", len(receivers)))
	return nil
}
