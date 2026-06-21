-- migrations/000006_add_profile_fields.down.sql
-- Reverses 000006_add_profile_fields.up.sql

DROP INDEX IF EXISTS idx_users_username;

ALTER TABLE users
    DROP COLUMN IF EXISTS username,
    DROP COLUMN IF EXISTS display_name,
    DROP COLUMN IF EXISTS bio,
    DROP COLUMN IF EXISTS avatar_url,
    DROP COLUMN IF EXISTS website,
    DROP COLUMN IF EXISTS location,
    DROP COLUMN IF EXISTS agency_name,
    DROP COLUMN IF EXISTS languages,
    DROP COLUMN IF EXISTS show_phone,
    DROP COLUMN IF EXISTS show_whatsapp,
    DROP COLUMN IF EXISTS notify_email,
    DROP COLUMN IF EXISTS preferred_lang;
