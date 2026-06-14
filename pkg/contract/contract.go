// Package contract defines the wire contract shared between the API service
// (publisher) and the notifications service (consumer). It contains only pure,
// JSON-serializable types — no domain models, no template engine, no transport
// client — so both independently-built modules can agree on the same payloads.
package contract

import "encoding/json"

// Channel is the Redis Pub/Sub channel notification jobs are published to.
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
