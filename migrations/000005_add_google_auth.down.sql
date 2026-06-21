-- migrations/000005_add_google_auth.down.sql
-- Reverses 000005_add_google_auth.up.sql

DROP INDEX IF EXISTS idx_users_google_id;

ALTER TABLE users
    DROP COLUMN IF EXISTS google_id,
    DROP COLUMN IF EXISTS auth_provider;

ALTER TABLE users
    ALTER COLUMN password_hash SET NOT NULL;
