// Package kafka builds the API service's Kafka producer and ensures the topics
// it publishes to exist. It is the API side of the notifications transport: the
// publisher writes contract.Message envelopes here and the notifications service
// consumes them.
package kafka

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"

	"se-school/internal/config"

	"github.com/segmentio/kafka-go"
)

// NewWriter builds a producer for cfg.Topic. RequireAll (acks=all) makes a
// successful write durable across the cluster, and the Hash balancer routes by
// message Key so all copies of the same notification land on one partition.
func NewWriter(cfg *config.Kafka) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers(cfg)...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
	}
}

// EnsureTopics creates the main and dead-letter topics if they are absent, so
// behaviour is deterministic instead of depending on broker auto-creation. It
// is a no-op when the topics already exist.
func EnsureTopics(ctx context.Context, cfg *config.Kafka) error {
	addrs := brokers(cfg)
	conn, err := kafka.DialContext(ctx, "tcp", addrs[0])
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}

	ctrlConn, err := kafka.DialContext(ctx, "tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer func() { _ = ctrlConn.Close() }()

	err = ctrlConn.CreateTopics(
		kafka.TopicConfig{Topic: cfg.Topic, NumPartitions: 1, ReplicationFactor: 1},
		kafka.TopicConfig{Topic: cfg.DLQTopic, NumPartitions: 1, ReplicationFactor: 1},
	)
	if err != nil && !errors.Is(err, kafka.TopicAlreadyExists) {
		return err
	}
	return nil
}

// brokers splits the comma-separated broker list into trimmed addresses.
func brokers(cfg *config.Kafka) []string {
	parts := strings.Split(cfg.Brokers, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
