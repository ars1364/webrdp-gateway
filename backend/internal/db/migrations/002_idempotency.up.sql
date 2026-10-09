-- One stored response per (user, Idempotency-Key). status 0 = in progress.
-- Rows expire after 24 h and are deleted by the reaper.
CREATE TABLE idempotency_keys (
    user_id       UUID NOT NULL REFERENCES users(id),
    key           UUID NOT NULL,
    request_hash  TEXT NOT NULL,
    status        INT NOT NULL DEFAULT 0,
    body          BYTEA NOT NULL DEFAULT ''::bytea,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at    TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, key)
);
CREATE INDEX idempotency_keys_expires ON idempotency_keys (expires_at);

-- Audit is append-only at the database level too.
CREATE FUNCTION audit_log_append_only() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        RAISE EXCEPTION 'audit_log is append-only';
    END IF;
    RETURN OLD; -- DELETE allowed only for retention (reaper)
END $$;
CREATE TRIGGER audit_log_no_update BEFORE UPDATE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_append_only();
