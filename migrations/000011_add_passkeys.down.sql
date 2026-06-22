-- migrations/000011_add_passkeys.down.sql

ALTER TABLE users
    DROP COLUMN IF EXISTS passkeys_enabled;

DROP TABLE IF EXISTS user_passkeys;
