-- migrations/000012_add_passkey_challenges.up.sql
-- Server-side WebAuthn challenge/session storage
-- Required for safe passkey registration and login flows

CREATE TABLE passkey_challenges (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    flow_id       TEXT NOT NULL UNIQUE,
    user_id       UUID REFERENCES users(id) ON DELETE CASCADE,
    flow_type     TEXT NOT NULL, -- register | login
    session_data  JSONB NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_passkey_challenges_flow_id    ON passkey_challenges(flow_id);
CREATE INDEX idx_passkey_challenges_user_id    ON passkey_challenges(user_id);
CREATE INDEX idx_passkey_challenges_expires_at ON passkey_challenges(expires_at);
