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

// RepliesTopic is the topic the notifications service publishes saga replies to
// and the API's saga orchestrator consumes. A reply reports the outcome of a
// saga command (a confirmation email dispatch) back to the orchestrator so it
// can complete or compensate the distributed transaction.
const RepliesTopic = "notifications.replies"

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
	// SagaID correlates a message that is part of an orchestrated saga with its
	// reply on RepliesTopic. It is empty for fire-and-forget notifications (e.g.
	// the cron repository-update alerts), for which the consumer sends no reply.
	SagaID string `json:"sagaId,omitempty"`
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

// ReplyStatus is the outcome the notifications service reports for a saga
// command it consumed.
type ReplyStatus = string

const (
	// ReplyDispatched means the email was successfully delivered to the SMTP
	// server (the notifier's local transaction committed as SENT).
	ReplyDispatched ReplyStatus = "dispatched"
	// ReplyFailed means the notifier could not dispatch the email and gave up
	// (terminal failure or retries exhausted); the orchestrator must compensate.
	ReplyFailed ReplyStatus = "failed"
)

// Reply is the envelope the notifications service publishes to RepliesTopic to
// report the outcome of a saga command back to the orchestrator. It is keyed by
// SagaID so the orchestrator can load the corresponding saga instance.
type Reply struct {
	SagaID         string      `json:"sagaId"`
	IdempotencyKey string      `json:"idempotencyKey"`
	Recipient      string      `json:"recipient"`
	Status         ReplyStatus `json:"status"`
	Reason         string      `json:"reason,omitempty"`
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
