CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        TEXT NOT NULL,
    password_hash   TEXT NOT NULL,
    totp_secret_enc BYTEA NOT NULL,
    role            TEXT NOT NULL DEFAULT 'admin' CHECK (role IN ('admin', 'user')),
    failed_logins   INT NOT NULL DEFAULT 0,
    locked_until    TIMESTAMPTZ NULL,
    last_totp_step  BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_deleted      BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at      TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX users_username_live ON users (lower(username)) WHERE NOT is_deleted;

CREATE TABLE sessions (
    token_hash  BYTEA PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id),
    ip          TEXT NOT NULL,
    user_agent  TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX sessions_user ON sessions (user_id);

CREATE TABLE connections (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id),
    name          TEXT NOT NULL,
    host          TEXT NOT NULL,
    port          INT NOT NULL CHECK (port BETWEEN 1 AND 65535),
    username      TEXT NOT NULL DEFAULT '',
    domain        TEXT NOT NULL DEFAULT '',
    password_enc  BYTEA NULL,
    security      TEXT NOT NULL DEFAULT 'any' CHECK (security IN ('any', 'nla', 'tls', 'rdp')),
    ignore_cert   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by    UUID NOT NULL REFERENCES users(id),
    is_deleted    BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at    TIMESTAMPTZ NULL
);
CREATE INDEX connections_user ON connections (user_id) WHERE NOT is_deleted;

CREATE TABLE audit_log (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID NULL REFERENCES users(id),
    action      TEXT NOT NULL,
    target      TEXT NOT NULL DEFAULT '',
    ip          TEXT NOT NULL DEFAULT '',
    detail      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_created ON audit_log (created_at DESC);
