-- migrations/000004_create_listing_media.up.sql
-- Photos for listings
-- Stored in Cloudflare R2, referenced here by URL + storage key
-- ADR-032: handlers never touch storage directly
-- Session 003 — vallescentrales schema foundation

CREATE TABLE listing_media (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id      UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,

    -- R2 storage reference
    url             TEXT NOT NULL,
    storage_key     TEXT NOT NULL UNIQUE,

    -- Display
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    alt_text        TEXT,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_media_listing_id ON listing_media(listing_id);
CREATE INDEX idx_media_is_primary  ON listing_media(listing_id, is_primary);
CREATE INDEX idx_media_sort_order  ON listing_media(listing_id, sort_order);
