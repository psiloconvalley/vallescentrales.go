-- migrations/000011_add_passkeys.up.sql
-- WebAuthn/FIDO2 passkey support
-- Stores public key credentials for passwordless authentication
-- One user can have multiple passkeys (phone + laptop + hardware key)

CREATE TABLE user_passkeys (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id   BYTEA NOT NULL UNIQUE,
    public_key      BYTEA NOT NULL,
    attestation_type TEXT NOT NULL DEFAULT 'none',
    aaguid          BYTEA,
    sign_count      BIGINT NOT NULL DEFAULT 0,
    device_name     TEXT,
    transports      TEXT[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at    TIMESTAMPTZ
);

CREATE INDEX idx_passkeys_user_id       ON user_passkeys(user_id);
CREATE INDEX idx_passkeys_credential_id ON user_passkeys(credential_id);

-- Track whether user has passkeys enabled
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS passkeys_enabled BOOLEAN NOT NULL DEFAULT FALSE;
