-- Orchestrated-saga support for the Subscribe flow.
--
-- saga_instances is the durable state machine of the "create subscription +
-- dispatch confirmation email" distributed transaction. outbox is the
-- transactional outbox: the confirmation-email command is written here inside
-- the same transaction as the subscription + saga rows, then published to Kafka
-- by a relay — making "state changed" and "command emitted" atomic and closing
-- the previous dual-write. Ids are UUID strings stored as TEXT (generated in Go)
-- to keep string scanning/params driver-agnostic.

CREATE TABLE IF NOT EXISTS saga_instances (
    id              TEXT        PRIMARY KEY,
    type            TEXT        NOT NULL DEFAULT 'subscribe_confirm',
    state           TEXT        NOT NULL,   -- AWAITING_NOTIFICATION|COMPLETED|COMPENSATING|COMPENSATED|FAILED
    subscription_id BIGINT      NOT NULL,
    email           TEXT        NOT NULL,
    idempotency_key TEXT        NOT NULL,
    attempts        INT         NOT NULL DEFAULT 0,
    last_error      TEXT        NOT NULL DEFAULT '',
    deadline_at     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_saga_state_deadline ON saga_instances (state, deadline_at);
CREATE INDEX IF NOT EXISTS idx_saga_subscription   ON saga_instances (subscription_id);

CREATE TABLE IF NOT EXISTS outbox (
    id           TEXT        PRIMARY KEY,
    saga_id      TEXT,
    topic        TEXT        NOT NULL,
    kafka_key    TEXT        NOT NULL,   -- = idempotency key (partition affinity)
    payload      BYTEA       NOT NULL,   -- marshaled contract.Message
    published_at TIMESTAMPTZ,            -- NULL = not yet published
    attempts     INT         NOT NULL DEFAULT 0,
    last_error   TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox (created_at) WHERE published_at IS NULL;
