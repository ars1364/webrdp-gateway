-- Server-side session recordings (guacd protocol dumps, played in-browser).
CREATE TABLE recordings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    label       TEXT NOT NULL DEFAULT '',
    target      TEXT NOT NULL,
    size_bytes  BIGINT NOT NULL DEFAULT 0 CHECK (size_bytes >= 0),
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at    TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by  UUID NOT NULL REFERENCES users(id),
    is_deleted  BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at  TIMESTAMPTZ NULL,
    CHECK (ended_at IS NULL OR ended_at >= started_at)
);
CREATE INDEX recordings_user ON recordings (user_id, started_at DESC) WHERE NOT is_deleted;
