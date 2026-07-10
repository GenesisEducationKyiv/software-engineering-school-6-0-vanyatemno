package models

import "time"

// OutboxMessage is persisted atomically with the state change that produced it,
// then published by the relay. PublishedAt is nil until published to the broker.
type OutboxMessage struct {
	ID          string
	SagaID      string
	Topic       string
	KafkaKey    string
	Payload     []byte
	PublishedAt *time.Time
	Attempts    int
	LastError   string
	CreatedAt   time.Time
}
