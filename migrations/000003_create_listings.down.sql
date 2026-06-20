-- migrations/000003_create_listings.down.sql
-- Reverses 000003_create_listings.up.sql

DROP TRIGGER IF EXISTS set_listings_updated_at ON listings;
DROP TABLE IF EXISTS listings;
DROP TYPE IF EXISTS listing_status;
DROP TYPE IF EXISTS property_type;
