package contract

import (
	"encoding/json"
	"testing"
)

func TestIdempotencyKey_Deterministic(t *testing.T) {
	payload := json.RawMessage(`{"code":"abc","link":"https://x/confirm/abc"}`)

	a := IdempotencyKey(Confirmation, payload)
	b := IdempotencyKey(Confirmation, payload)

	if a == "" {
		t.Fatal("expected a non-empty key")
	}
	if a != b {
		t.Fatalf("expected deterministic key, got %q vs %q", a, b)
	}
}

func TestIdempotencyKey_VariesByTemplateAndPayload(t *testing.T) {
	p1 := json.RawMessage(`{"code":"abc"}`)
	p2 := json.RawMessage(`{"code":"xyz"}`)

	if IdempotencyKey(Confirmation, p1) == IdempotencyKey(Confirmation, p2) {
		t.Fatal("different payloads should produce different keys")
	}
	if IdempotencyKey(Confirmation, p1) == IdempotencyKey(RepositoryUpdated, p1) {
		t.Fatal("different templates should produce different keys")
	}
}
