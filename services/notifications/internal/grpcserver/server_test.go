package grpcserver

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ghnotify/contract"
	"ghnotify/contract/notifierpb"
	"ghnotify/notifier/internal/dispatch"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type senderFake struct {
	err   error
	calls int
	last  dispatch.Command
}

func (s *senderFake) Send(_ context.Context, cmd dispatch.Command) error {
	s.calls++
	s.last = cmd
	return s.err
}

func req() *notifierpb.NotifyRequest {
	return &notifierpb.NotifyRequest{
		Template:       contract.RepositoryUpdated,
		Recipient:      "a@x.com",
		Payload:        []byte(`{"name":"r"}`),
		IdempotencyKey: "k1",
	}
}

func TestNotify_Success_ReturnsOk(t *testing.T) {
	s := New(&senderFake{})

	resp, err := s.Notify(context.Background(), req())
	if err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if !resp.GetOk() {
		t.Fatal("expected ok=true")
	}
}

func TestNotify_DeliveryFailure_ReturnsNotOkWithReason(t *testing.T) {
	s := New(&senderFake{err: errors.New("smtp refused")})

	resp, err := s.Notify(context.Background(), req())
	if err != nil {
		t.Fatalf("expected no gRPC error for a delivery failure, got %v", err)
	}
	if resp.GetOk() {
		t.Fatal("expected ok=false")
	}
	if !strings.Contains(resp.GetReason(), "smtp refused") {
		t.Fatalf("reason = %q, want it to contain the cause", resp.GetReason())
	}
}

func TestNotify_ForwardsCommand(t *testing.T) {
	sf := &senderFake{}
	s := New(sf)

	if _, err := s.Notify(context.Background(), req()); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if sf.last.Recipient != "a@x.com" || sf.last.Template != contract.RepositoryUpdated || sf.last.IdempotencyKey != "k1" {
		t.Fatalf("command not forwarded verbatim: %+v", sf.last)
	}
}

func TestNotify_EmptyRecipient_InvalidArgument(t *testing.T) {
	sf := &senderFake{}
	s := New(sf)
	r := req()
	r.Recipient = ""

	_, err := s.Notify(context.Background(), r)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
	if sf.calls != 0 {
		t.Fatalf("sender must not be called for an invalid request; calls = %d", sf.calls)
	}
}
