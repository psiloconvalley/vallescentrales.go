-- migrations/000008_add_payments.up.sql
-- Payment transaction foundation
-- Supports fiat (Stripe/Conekta) and crypto (USDC/ETH/BTC/SOL)

CREATE TYPE payment_method AS ENUM (
    'stripe',
    'conekta',
    'usdc',
    'eth',
    'btc',
    'sol',
    'bank_transfer'
);

CREATE TYPE payment_status AS ENUM (
    'pending',
    'processing',
    'confirmed',
    'failed',
    'refunded',
    'disputed'
);

CREATE TYPE payment_purpose AS ENUM (
    'featured_listing',
    'agent_subscription',
    'property_deposit',
    'property_purchase'
);

CREATE TABLE payments (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id          UUID REFERENCES listings(id) ON DELETE SET NULL,
    payer_id            UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    payee_id            UUID REFERENCES users(id) ON DELETE SET NULL,

    -- Amount
    amount_mxn          NUMERIC(20, 2) NOT NULL CHECK (amount_mxn > 0),
    amount_usd          NUMERIC(20, 8),
    amount_crypto       NUMERIC(36, 18),

    -- Method
    method              payment_method NOT NULL,
    purpose             payment_purpose NOT NULL,
    status              payment_status NOT NULL DEFAULT 'pending',

    -- Fiat references
    stripe_payment_id   TEXT UNIQUE,
    conekta_order_id    TEXT UNIQUE,

    -- Crypto references
    crypto_token        TEXT,
    crypto_network      TEXT,
    crypto_tx_hash      TEXT UNIQUE,
    crypto_from_address TEXT,
    crypto_to_address   TEXT,
    crypto_block_number BIGINT,
    crypto_confirmed_at TIMESTAMPTZ,

    -- Metadata
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_listing_id  ON payments(listing_id);
CREATE INDEX idx_payments_payer_id    ON payments(payer_id);
CREATE INDEX idx_payments_payee_id    ON payments(payee_id);
CREATE INDEX idx_payments_status      ON payments(status);
CREATE INDEX idx_payments_method      ON payments(method);
CREATE INDEX idx_payments_created_at  ON payments(created_at DESC);

CREATE TRIGGER set_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
