-- migrations/000005_add_google_auth.up.sql
-- Add Google OAuth support to users table
-- password_hash becomes nullable for Google-only users
-- auth_provider tracks registration method

ALTER TABLE users
    ALTER COLUMN password_hash DROP NOT NULL;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS google_id     TEXT UNIQUE,
    ADD COLUMN IF NOT EXISTS auth_provider TEXT NOT NULL DEFAULT 'email';

CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);
