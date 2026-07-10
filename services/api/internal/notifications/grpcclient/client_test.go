package grpcclient

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ghnotify/contract"
	"ghnotify/contract/notifierpb"

	"google.golang.org/grpc"
)

type rpcFake struct {
	resp *notifierpb.NotifyResponse
	err  error
	last *notifierpb.NotifyRequest
}

func (f *rpcFake) Notify(_ context.Context, in *notifierpb.NotifyRequest, _ ...grpc.CallOption) (*notifierpb.NotifyResponse, error) {
	f.last = in
	return f.resp, f.err
}

func newTestClient(rpc notifierpb.NotifierClient) *Client {
	return &Client{rpc: rpc}
}

func payload() contract.RepositoryUpdateEmailPayload {
	return contract.RepositoryUpdateEmailPayload{Name: "r", Owner: "o", Version: "v2", UnsubscribeURL: "u"}
}

func TestNotify_Ok_ReturnsNilAndStampsDeterministicKey(t *testing.T) {
	fake := &rpcFake{resp: &notifierpb.NotifyResponse{Ok: true}}
	c := newTestClient(fake)

	data := payload()
	if err := c.Notify(context.Background(), "a@x.com", contract.RepositoryUpdated, data); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if fake.last.GetRecipient() != "a@x.com" || fake.last.GetTemplate() != contract.RepositoryUpdated {
		t.Fatalf("unexpected request: %+v", fake.last)
	}
	wantKey := contract.IdempotencyKey(contract.RepositoryUpdated, fake.last.GetPayload())
	if fake.last.GetIdempotencyKey() != wantKey {
		t.Fatalf("idempotency key = %q, want %q", fake.last.GetIdempotencyKey(), wantKey)
	}
}

func TestNotify_NotOk_ReturnsError(t *testing.T) {
	fake := &rpcFake{resp: &notifierpb.NotifyResponse{Ok: false, Reason: "mailbox full"}}
	c := newTestClient(fake)

	err := c.Notify(context.Background(), "a@x.com", contract.RepositoryUpdated, payload())
	if err == nil {
		t.Fatal("expected error when notifier reports ok=false")
	}
	if !strings.Contains(err.Error(), "mailbox full") {
		t.Fatalf("error = %q, want it to carry the reason", err.Error())
	}
}

func TestNotify_TransportError_ReturnsError(t *testing.T) {
	fake := &rpcFake{err: errors.New("connection refused")}
	c := newTestClient(fake)

	err := c.Notify(context.Background(), "a@x.com", contract.RepositoryUpdated, payload())
	if err == nil {
		t.Fatal("expected error on transport failure")
	}
}
