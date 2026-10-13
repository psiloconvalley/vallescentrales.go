-- migrations/000013_add_listing_essentials.down.sql
DROP INDEX IF EXISTS idx_listings_legal_regime;
DROP INDEX IF EXISTS idx_listings_operation_type;

ALTER TABLE listings
    DROP COLUMN IF EXISTS legal_regime,
    DROP COLUMN IF EXISTS operation_type;

DROP TYPE IF EXISTS legal_regime;
DROP TYPE IF EXISTS operation_type;
