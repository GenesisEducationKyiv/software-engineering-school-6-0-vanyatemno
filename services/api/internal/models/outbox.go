package models

import "time"

// OutboxMessage is a row of the transactional outbox: a Kafka message persisted
// atomically with the state change that produced it, later published by the
// relay. PublishedAt is nil until the relay has written it to the broker.
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
