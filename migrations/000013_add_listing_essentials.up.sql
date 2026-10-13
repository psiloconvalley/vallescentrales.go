-- migrations/000013_add_listing_essentials.up.sql
-- Add operation_type and legal_regime to the listings table.
-- These are the two most important fields for Oaxacan real estate:
--   operation_type: Is this listing for sale or rent?
--   legal_regime:   What is the legal status of the land?
--                   (Escritura Pública / Ejidal / Comunal / Posesión)

CREATE TYPE operation_type AS ENUM (
    'sale',
    'rent'
);

CREATE TYPE legal_regime AS ENUM (
    'escritura_publica',
    'ejidal',
    'comunal',
    'posesion'
);

ALTER TABLE listings
    ADD COLUMN IF NOT EXISTS operation_type operation_type NOT NULL DEFAULT 'sale',
    ADD COLUMN IF NOT EXISTS legal_regime   legal_regime   NOT NULL DEFAULT 'escritura_publica';

CREATE INDEX IF NOT EXISTS idx_listings_operation_type ON listings(operation_type);
CREATE INDEX IF NOT EXISTS idx_listings_legal_regime   ON listings(legal_regime);
