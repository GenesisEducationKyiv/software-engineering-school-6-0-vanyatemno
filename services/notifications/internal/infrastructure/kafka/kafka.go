// Package kafka builds the notifications service's Kafka consumer (reader) and
// the dead-letter producer. The reader is configured for manual offset commit so
// the worker acks a message only after it is handled or dead-lettered.
package kafka

import (
	"strings"

	"ghnotify/notifier/internal/config"

	"github.com/segmentio/kafka-go"
)

// NewReader builds a consumer-group reader. CommitInterval 0 disables periodic
// auto-commit, so the worker controls exactly when offsets advance.
func NewReader(cfg *config.Kafka) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers(cfg),
		GroupID:        cfg.GroupID,
		Topic:          cfg.Topic,
		CommitInterval: 0,
	})
}

// NewDLQWriter builds a producer for the dead-letter topic.
func NewDLQWriter(cfg *config.Kafka) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers(cfg)...),
		Topic:        cfg.DLQTopic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
	}
}

// NewReplyWriter builds a producer for the saga replies topic. Replies are keyed
// by saga id (Hash balancer) so a saga's replies keep per-partition order.
func NewReplyWriter(cfg *config.Kafka) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers(cfg)...),
		Topic:        cfg.RepliesTopic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
	}
}

// brokers splits the comma-separated broker list into trimmed addresses.
func brokers(cfg *config.Kafka) []string {
	parts := strings.Split(cfg.Brokers, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
