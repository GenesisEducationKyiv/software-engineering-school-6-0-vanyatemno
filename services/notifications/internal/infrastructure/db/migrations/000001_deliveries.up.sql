-- deliveries is the notifier's durable participant state in the distributed
-- transaction. Each row tracks one confirmation email's dispatch outcome; the
-- saga reply is derived from its terminal state. The UNIQUE (idempotency_key,
-- recipient) constraint is also the notifier's durable idempotency guard: a
-- redelivered command finds the existing row instead of sending twice.

CREATE TABLE IF NOT EXISTS deliveries (
    id              BIGSERIAL   PRIMARY KEY,
    saga_id         TEXT        NOT NULL,
    idempotency_key TEXT        NOT NULL,
    recipient       TEXT        NOT NULL,
    template        TEXT        NOT NULL,
    state           TEXT        NOT NULL,   -- SENDING|SENT|FAILED
    last_error      TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (idempotency_key, recipient)
);

CREATE INDEX IF NOT EXISTS idx_deliveries_saga ON deliveries (saga_id);
