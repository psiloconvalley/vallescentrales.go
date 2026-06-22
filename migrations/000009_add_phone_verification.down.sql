-- migrations/000009_add_phone_verification.down.sql

ALTER TABLE users
    DROP COLUMN IF EXISTS phone_verified;

DROP TABLE IF EXISTS verification_tokens;
