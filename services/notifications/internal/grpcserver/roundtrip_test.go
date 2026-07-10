package grpcserver

import (
	"context"
	"net"
	"testing"

	"ghnotify/contract"
	"ghnotify/contract/notifierpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestNotify_RealTransport_RoundTrip exercises the full wire the API uses: the
// generated client marshals a request over a real gRPC connection to the
// registered server, which decodes it and invokes the Sender. It guards against
// registration / proto-wiring mistakes the in-process handler tests can't catch.
func TestNotify_RealTransport_RoundTrip(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := grpc.NewServer()
	sf := &senderFake{}
	notifierpb.RegisterNotifierServer(srv, New(sf))
	go func() { _ = srv.Serve(lis) }()
	defer srv.GracefulStop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	client := notifierpb.NewNotifierClient(conn)
	resp, err := client.Notify(context.Background(), &notifierpb.NotifyRequest{
		Template:       contract.RepositoryUpdated,
		Recipient:      "a@x.com",
		Payload:        []byte(`{"name":"r"}`),
		IdempotencyKey: "k",
	})
	if err != nil {
		t.Fatalf("Notify over the wire: %v", err)
	}
	if !resp.GetOk() {
		t.Fatalf("ok = false, reason = %q", resp.GetReason())
	}
	if sf.calls != 1 || sf.last.Recipient != "a@x.com" || sf.last.Template != contract.RepositoryUpdated {
		t.Fatalf("server did not receive the decoded command: %+v", sf.last)
	}
}
