-- migrations/000007_add_currency_support.up.sql
-- Currency foundation — exchange rate cache and listing price display
-- Supports future crypto payment acceptance

-- Exchange rate cache — avoids hitting API on every request
CREATE TABLE exchange_rates (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_cur    TEXT NOT NULL,
    to_cur      TEXT NOT NULL,
    rate        NUMERIC(20, 8) NOT NULL,
    source      TEXT NOT NULL DEFAULT 'exchangerate-api',
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(from_cur, to_cur)
);

CREATE INDEX idx_exchange_rates_pair ON exchange_rates(from_cur, to_cur);
CREATE INDEX idx_exchange_rates_fetched ON exchange_rates(fetched_at DESC);

-- Add crypto acceptance to listings
ALTER TABLE listings
    ADD COLUMN IF NOT EXISTS accepts_crypto   BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS crypto_address   TEXT,
    ADD COLUMN IF NOT EXISTS crypto_tokens    TEXT[] NOT NULL DEFAULT '{}';
