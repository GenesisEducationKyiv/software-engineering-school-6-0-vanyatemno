// Package contract defines the wire contract shared between the API service
// (publisher) and the notifications service (consumer). It contains only pure,
// JSON-serializable types — no domain models, no template engine, no transport
// client — so both independently-built modules can agree on the same payloads.
package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Topic is the Kafka topic notification jobs are published to and consumed from.
const Topic = "notifications.events"

// DLQTopic is the dead-letter topic the consumer routes messages to when they
// fail terminally (bad payload, unknown template) or exhaust their retries.
const DLQTopic = "notifications.events.dlq"

// Channel is the legacy Redis Pub/Sub channel name.
//
// Deprecated: the transport is Kafka now — use Topic. Kept only until the
// publisher and consumer have migrated off it.
const Channel = "notifications:events"

// TemplateName identifies which email template the notifications service should
// render for a given message.
type TemplateName = string

const (
	Confirmation      TemplateName = "confirm"
	RepositoryUpdated TemplateName = "repository_update"
)

// Message is the envelope published to Channel. Payload carries the
// template-specific body (one of the *EmailPayload types below) and is decoded
// by the consumer based on Template.
type Message struct {
	Template  TemplateName    `json:"template"`
	Receivers []string        `json:"receivers"`
	Payload   json.RawMessage `json:"payload"`
	// IdempotencyKey deterministically identifies this notification so the
	// consumer can send each email at most once even if Kafka redelivers the
	// message or the producer publishes it twice. Set via IdempotencyKey().
	IdempotencyKey string `json:"idempotencyKey"`
}

// ConfirmEmailPayload is the body for the Confirmation template.
type ConfirmEmailPayload struct {
	Code string `json:"code"`
	Link string `json:"link"`
}

// RepositoryUpdateEmailPayload is the body for the RepositoryUpdated template.
type RepositoryUpdateEmailPayload struct {
	Name           string `json:"name"`
	Owner          string `json:"owner"`
	Version        string `json:"version"`
	UnsubscribeURL string `json:"unsubscribeUrl"`
}

// IdempotencyKey derives a deterministic key from a message's template and
// payload. The same logical notification always maps to the same key, letting
// the consumer deduplicate redelivered or re-published messages. The payloads
// above carry no timestamps or nonces, so the hash is stable across publishes.
func IdempotencyKey(template TemplateName, payload json.RawMessage) string {
	h := sha256.New()
	h.Write([]byte(template))
	h.Write([]byte{0})
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
