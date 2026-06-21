-- migrations/000006_add_profile_fields.up.sql
-- Extended user profile fields
-- Supports public profile pages, avatar, username, bio, preferences

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS username          TEXT UNIQUE,
    ADD COLUMN IF NOT EXISTS display_name      TEXT,
    ADD COLUMN IF NOT EXISTS bio               TEXT,
    ADD COLUMN IF NOT EXISTS avatar_url        TEXT,
    ADD COLUMN IF NOT EXISTS website           TEXT,
    ADD COLUMN IF NOT EXISTS location          TEXT,
    ADD COLUMN IF NOT EXISTS agency_name       TEXT,
    ADD COLUMN IF NOT EXISTS languages         TEXT[] NOT NULL DEFAULT '{"es"}',
    ADD COLUMN IF NOT EXISTS show_phone        BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS show_whatsapp     BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS notify_email      BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS preferred_lang    TEXT NOT NULL DEFAULT 'es';

-- Username index for fast lookup by public profile URL
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
