CREATE TABLE IF NOT EXISTS repositories (
    id          BIGSERIAL PRIMARY KEY,
    owner       TEXT        NOT NULL,
    name        TEXT        NOT NULL,
    version     TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_repository           ON repositories (owner, name);
CREATE INDEX IF NOT EXISTS idx_repositories_deleted ON repositories (deleted_at);

CREATE TABLE IF NOT EXISTS codes (
    id          BIGSERIAL PRIMARY KEY,
    code        TEXT        NOT NULL UNIQUE,
    type        TEXT        NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_codes_type    ON codes (type);
CREATE INDEX IF NOT EXISTS idx_codes_deleted ON codes (deleted_at);

CREATE TABLE IF NOT EXISTS subscriptions (
    id                   BIGSERIAL PRIMARY KEY,
    repository_id        BIGINT      NOT NULL,
    subscribe_code_id    BIGINT      NOT NULL,
    unsubscribe_code_id  BIGINT      NOT NULL,
    email                TEXT        NOT NULL,
    is_confirmed         BOOLEAN     NOT NULL DEFAULT FALSE,
    last_seen_tag        TEXT        NOT NULL DEFAULT '',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_deleted ON subscriptions (deleted_at);

CREATE UNIQUE INDEX IF NOT EXISTS unique_idx_email_repository_id
    ON subscriptions (email, repository_id)
    WHERE deleted_at IS NULL;
