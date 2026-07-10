// Package contract holds the JSON wire types shared by the API (publisher) and
// notifications (consumer) services.
package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const Topic = "notifications.events"

const DLQTopic = "notifications.events.dlq"

const RepliesTopic = "notifications.replies"

type TemplateName = string

const (
	Confirmation      TemplateName = "confirm"
	RepositoryUpdated TemplateName = "repository_update"
)

type Message struct {
	Template  TemplateName    `json:"template"`
	Receivers []string        `json:"receivers"`
	Payload   json.RawMessage `json:"payload"`
	// IdempotencyKey lets the consumer send each email at most once across
	// redeliveries. Set via IdempotencyKey().
	IdempotencyKey string `json:"idempotencyKey"`
	// SagaID correlates a saga message with its reply on RepliesTopic; empty for
	// fire-and-forget notifications, which get no reply.
	SagaID string `json:"sagaId,omitempty"`
}

type ConfirmEmailPayload struct {
	Code string `json:"code"`
	Link string `json:"link"`
}

type RepositoryUpdateEmailPayload struct {
	Name           string `json:"name"`
	Owner          string `json:"owner"`
	Version        string `json:"version"`
	UnsubscribeURL string `json:"unsubscribeUrl"`
}

type ReplyStatus = string

const (
	ReplyDispatched ReplyStatus = "dispatched"
	ReplyFailed     ReplyStatus = "failed"
)

type Reply struct {
	SagaID         string      `json:"sagaId"`
	IdempotencyKey string      `json:"idempotencyKey"`
	Recipient      string      `json:"recipient"`
	Status         ReplyStatus `json:"status"`
	Reason         string      `json:"reason,omitempty"`
}

// IdempotencyKey derives a stable key from a message's template and payload. The
// payloads carry no timestamps or nonces, so re-publishing the same notification
// yields the same key and the consumer can dedupe.
func IdempotencyKey(template TemplateName, payload json.RawMessage) string {
	h := sha256.New()
	h.Write([]byte(template))
	h.Write([]byte{0})
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
