-- migrations/000007_add_currency_support.down.sql

ALTER TABLE listings
    DROP COLUMN IF EXISTS accepts_crypto,
    DROP COLUMN IF EXISTS crypto_address,
    DROP COLUMN IF EXISTS crypto_tokens;

DROP TABLE IF EXISTS exchange_rates;
