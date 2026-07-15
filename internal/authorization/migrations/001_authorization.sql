CREATE TABLE IF NOT EXISTS authorization_sessions (
    ref TEXT PRIMARY KEY,
    subject TEXT NOT NULL,
    token_hash BYTEA NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS authorization_consent_grants (
    ref TEXT PRIMARY KEY,
    subject TEXT NOT NULL,
    audience TEXT NOT NULL,
    scopes TEXT[] NOT NULL,
    revision BIGINT NOT NULL CHECK (revision > 0),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS authorization_audit_events (
    sequence BIGSERIAL PRIMARY KEY,
    session_ref TEXT NOT NULL DEFAULT '',
    grant_ref TEXT NOT NULL DEFAULT '',
    scope TEXT NOT NULL,
    decision TEXT NOT NULL CHECK (decision IN ('allow', 'deny')),
    revision BIGINT NOT NULL CHECK (revision >= 0)
);

CREATE INDEX IF NOT EXISTS authorization_consent_grants_subject_audience_idx
    ON authorization_consent_grants (subject, audience);
