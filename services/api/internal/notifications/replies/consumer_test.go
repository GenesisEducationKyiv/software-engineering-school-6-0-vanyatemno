package replies

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"ghnotify/contract"

	"github.com/segmentio/kafka-go"
)

// readerMock yields queued messages once, then returns an error carrying a
// canceled context so Run exits cleanly.
type readerMock struct {
	msgs      []kafka.Message
	idx       int
	committed int
	cancel    context.CancelFunc
}

func (r *readerMock) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if r.idx >= len(r.msgs) {
		r.cancel() // no more input — cancel so Run returns
		return kafka.Message{}, errors.New("no more messages")
	}
	m := r.msgs[r.idx]
	r.idx++
	return m, nil
}

func (r *readerMock) CommitMessages(_ context.Context, msgs ...kafka.Message) error {
	r.committed += len(msgs)
	return nil
}

type handlerMock struct {
	handled []contract.Reply
	err     error
}

func (h *handlerMock) HandleReply(_ context.Context, reply contract.Reply) error {
	h.handled = append(h.handled, reply)
	return h.err
}

func mustReply(t *testing.T, r contract.Reply) kafka.Message {
	t.Helper()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return kafka.Message{Key: []byte(r.SagaID), Value: b}
}

func TestRun_HandlesReplyAndCommits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := &readerMock{cancel: cancel, msgs: []kafka.Message{
		mustReply(t, contract.Reply{SagaID: "s1", Status: contract.ReplyDispatched}),
	}}
	handler := &handlerMock{}

	_ = New(reader, handler).Run(ctx)

	if len(handler.handled) != 1 || handler.handled[0].SagaID != "s1" {
		t.Fatalf("expected reply s1 handled, got %v", handler.handled)
	}
	if reader.committed != 1 {
		t.Fatalf("expected 1 commit, got %d", reader.committed)
	}
}

func TestRun_PoisonMessage_IsCommittedNotHandled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := &readerMock{cancel: cancel, msgs: []kafka.Message{
		{Value: []byte("not json")},
	}}
	handler := &handlerMock{}

	_ = New(reader, handler).Run(ctx)

	if len(handler.handled) != 0 {
		t.Fatalf("expected poison message not handled, got %v", handler.handled)
	}
	if reader.committed != 1 {
		t.Fatalf("expected poison message committed (advanced), got %d", reader.committed)
	}
}

func TestRun_HandlerError_StillCommits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := &readerMock{cancel: cancel, msgs: []kafka.Message{
		mustReply(t, contract.Reply{SagaID: "s2", Status: contract.ReplyFailed}),
	}}
	handler := &handlerMock{err: errors.New("db blip")}

	_ = New(reader, handler).Run(ctx)

	if reader.committed != 1 {
		t.Fatalf("expected commit even on handler error (sweeper is the backstop), got %d", reader.committed)
	}
}
