-- Per-connection display quality (bandwidth profile). Additive + defaulted,
-- so old app versions keep working (forward-compatible).
ALTER TABLE connections
    ADD COLUMN quality TEXT NOT NULL DEFAULT 'balanced'
    CHECK (quality IN ('high', 'balanced', 'low'));
