package relay

import (
	"context"
	"errors"
	"testing"

	"se-school/internal/models"

	"github.com/segmentio/kafka-go"
)

type storeMock struct {
	unpublished []*models.OutboxMessage
	published   []string
	failed      []string
	getErr      error
}

func (s *storeMock) GetUnpublished(_ context.Context, _ int) ([]*models.OutboxMessage, error) {
	return s.unpublished, s.getErr
}
func (s *storeMock) MarkPublished(_ context.Context, id string) error {
	s.published = append(s.published, id)
	return nil
}
func (s *storeMock) MarkFailed(_ context.Context, id, _ string) error {
	s.failed = append(s.failed, id)
	return nil
}

type producerMock struct {
	written [][]byte
	err     error
}

func (p *producerMock) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	if p.err != nil {
		return p.err
	}
	for _, m := range msgs {
		p.written = append(p.written, m.Value)
	}
	return nil
}

func TestDrain_PublishesAndMarksPublished(t *testing.T) {
	store := &storeMock{unpublished: []*models.OutboxMessage{
		{ID: "a", KafkaKey: "k1", Payload: []byte(`{"x":1}`)},
		{ID: "b", KafkaKey: "k2", Payload: []byte(`{"x":2}`)},
	}}
	prod := &producerMock{}
	r := New(store, prod, 0, 10)

	r.drain(context.Background())

	if len(prod.written) != 2 {
		t.Fatalf("expected 2 published messages, got %d", len(prod.written))
	}
	if len(store.published) != 2 {
		t.Fatalf("expected 2 marked published, got %v", store.published)
	}
	if len(store.failed) != 0 {
		t.Fatalf("expected no failures, got %v", store.failed)
	}
}

func TestDrain_PublishError_MarksFailedNotPublished(t *testing.T) {
	store := &storeMock{unpublished: []*models.OutboxMessage{{ID: "a", KafkaKey: "k1", Payload: []byte(`{}`)}}}
	prod := &producerMock{err: errors.New("broker down")}
	r := New(store, prod, 0, 10)

	r.drain(context.Background())

	if len(store.published) != 0 {
		t.Fatalf("expected nothing marked published, got %v", store.published)
	}
	if len(store.failed) != 1 {
		t.Fatalf("expected 1 marked failed, got %v", store.failed)
	}
}
