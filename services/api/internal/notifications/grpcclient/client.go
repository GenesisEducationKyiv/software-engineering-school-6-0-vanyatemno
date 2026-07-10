// Package grpcclient is the API side of the synchronous notifications boundary.
// It satisfies the repository service's Notifier seam by calling the notifier's
// gRPC Notify RPC once per recipient and waiting for the outcome, so the caller
// can advance a subscription's last_seen_tag only when delivery is confirmed.
//
// It replaces the fire-and-forget Kafka publisher for the cron repository-update
// path; the subscribe-confirmation saga still flows over Kafka.
package grpcclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ghnotify/contract"
	"ghnotify/contract/notifierpb"

	"google.golang.org/grpc"
)

// Client calls the notifier's synchronous delivery API.
type Client struct {
	rpc     notifierpb.NotifierClient
	timeout time.Duration
}

// New wraps an established gRPC connection. timeout (when > 0) bounds each Notify
// call; the connection's lifecycle is owned by the caller.
func New(conn grpc.ClientConnInterface, timeout time.Duration) *Client {
	return &Client{rpc: notifierpb.NewNotifierClient(conn), timeout: timeout}
}

// Notify delivers one email to one recipient synchronously. It returns nil only
// when the notifier confirms delivery (ok). A transport error or a not-ok
// response is returned as an error so the caller declines to advance
// last_seen_tag and the subscriber is retried on the next cron run.
func (c *Client) Notify(ctx context.Context, recipient string, template contract.TemplateName, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal notification payload: %w", err)
	}
	// Deterministic key: the notifier dedupes per (key, recipient), so a retried
	// send of the same version after a lost response does not re-email.
	key := contract.IdempotencyKey(template, payload)

	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	resp, err := c.rpc.Notify(ctx, &notifierpb.NotifyRequest{
		Template:       template,
		Recipient:      recipient,
		Payload:        payload,
		IdempotencyKey: key,
	})
	if err != nil {
		return fmt.Errorf("notify %s: %w", recipient, err)
	}
	if !resp.GetOk() {
		return fmt.Errorf("notify %s: notifier reported failure: %s", recipient, resp.GetReason())
	}
	return nil
}
