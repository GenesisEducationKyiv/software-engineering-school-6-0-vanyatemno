// Package kafka builds the API's Kafka producer and reply reader and ensures its
// topics exist — the API side of the notifications transport.
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

// RequireAll (acks=all) makes writes durable across the cluster; the Hash
// balancer routes by message Key so copies of a notification share a partition.
func NewWriter(cfg *config.Kafka) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers(cfg)...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
	}
}

// EnsureTopics creates the topics up front so behaviour doesn't depend on broker
// auto-creation. No-op when they already exist.
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
		kafka.TopicConfig{Topic: cfg.RepliesTopic, NumPartitions: 1, ReplicationFactor: 1},
	)
	if err != nil && !errors.Is(err, kafka.TopicAlreadyExists) {
		return err
	}
	return nil
}

// CommitInterval 0 disables periodic auto-commit so the reply consumer acks a
// reply only after the orchestrator has handled it.
func NewReplyReader(cfg *config.Kafka) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers(cfg),
		GroupID:        cfg.SagaGroupID,
		Topic:          cfg.RepliesTopic,
		CommitInterval: 0,
	})
}

func brokers(cfg *config.Kafka) []string {
	parts := strings.Split(cfg.Brokers, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
