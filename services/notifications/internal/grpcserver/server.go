// Package grpcserver exposes the notifier's synchronous delivery API over gRPC.
// The API service calls Notify once per subscription on a repository update and
// blocks on the outcome, so it can advance the subscription's last_seen_tag only
// when the email was actually delivered. Delivery reuses the same at-most-once
// core (dispatch.Dispatcher) as the Kafka saga path.
package grpcserver

import (
	"context"

	"ghnotify/contract/notifierpb"
	"ghnotify/notifier/internal/dispatch"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Sender delivers one notification command at most once. It is satisfied by
// *dispatch.Dispatcher; the interface keeps the server testable.
type Sender interface {
	Send(ctx context.Context, cmd dispatch.Command) error
}

// Server implements notifierpb.NotifierServer by delegating to the shared
// delivery core.
type Server struct {
	notifierpb.UnimplementedNotifierServer
	sender Sender
}

func New(sender Sender) *Server {
	return &Server{sender: sender}
}

// Notify delivers one email to one recipient and reports the outcome
// synchronously. It returns:
//   - ok=true when the email was delivered (or had already been delivered — an
//     idempotent skip), so the caller may advance last_seen_tag;
//   - ok=false with a reason when delivery failed, so the caller leaves the tag
//     untouched and retries on the next cron run;
//   - a gRPC error only for a malformed request or a cancelled/expired context.
func (s *Server) Notify(ctx context.Context, req *notifierpb.NotifyRequest) (*notifierpb.NotifyResponse, error) {
	if req.GetRecipient() == "" {
		return nil, status.Error(codes.InvalidArgument, "recipient is required")
	}
	if req.GetTemplate() == "" {
		return nil, status.Error(codes.InvalidArgument, "template is required")
	}

	err := s.sender.Send(ctx, dispatch.Command{
		Template:       req.GetTemplate(),
		Recipient:      req.GetRecipient(),
		Payload:        req.GetPayload(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		// A cancelled/expired context is a transport-level condition, not a
		// delivery verdict — surface it as a gRPC error so the client retries.
		if ctx.Err() != nil {
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		zap.L().Warn("notify delivery failed",
			zap.String("recipient", req.GetRecipient()),
			zap.String("template", req.GetTemplate()),
			zap.Error(err))
		return &notifierpb.NotifyResponse{Ok: false, Reason: err.Error()}, nil
	}

	return &notifierpb.NotifyResponse{Ok: true}, nil
}
