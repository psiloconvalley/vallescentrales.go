-- migrations/000009_add_phone_verification.up.sql
-- Phone OTP verification
-- One active token per user at a time

CREATE TABLE verification_tokens (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT NOT NULL,
    type        TEXT NOT NULL DEFAULT 'phone',
    phone       TEXT,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes',
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_verification_tokens_user_id   ON verification_tokens(user_id);
CREATE INDEX idx_verification_tokens_token     ON verification_tokens(token);
CREATE INDEX idx_verification_tokens_expires   ON verification_tokens(expires_at);

-- Add phone_verified to users
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS phone_verified BOOLEAN NOT NULL DEFAULT FALSE;
