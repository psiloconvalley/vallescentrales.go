-- migrations/000003_create_listings.up.sql
-- Listings: the primary product of vallescentrales
-- ADR-004: required fields, status flow, MXN as source of truth
-- ADR-005: canonical vocabulary locked
-- Session 003 — vallescentrales schema foundation

CREATE TYPE listing_status AS ENUM (
    'draft',
    'active',
    'under_contract',
    'sold',
    'archived'
);

CREATE TYPE property_type AS ENUM (
    'land',
    'house',
    'rancho',
    'commercial',
    'cabin'
);

CREATE TABLE listings (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id            UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,

    -- Identity
    title               TEXT NOT NULL,
    slug                TEXT NOT NULL UNIQUE,
    description         TEXT,

    -- Classification
    property_type       property_type NOT NULL,
    status              listing_status NOT NULL DEFAULT 'draft',

    -- Price (MXN is source of truth — ADR-005)
    price_mxn           NUMERIC(14, 2) NOT NULL CHECK (price_mxn > 0),

    -- Size (optional at creation — ADR-004)
    area_m2             NUMERIC(10, 2) CHECK (area_m2 > 0),
    construction_m2     NUMERIC(10, 2) CHECK (construction_m2 > 0),

    -- Location hierarchy: municipality → community → coordinates
    municipality        TEXT NOT NULL,
    community           TEXT,

    -- Coordinates optional — map degrades gracefully (ADR-004)
    latitude            NUMERIC(10, 7),
    longitude           NUMERIC(10, 7),

    -- Engagement
    is_featured         BOOLEAN NOT NULL DEFAULT FALSE,
    view_count          INTEGER NOT NULL DEFAULT 0 CHECK (view_count >= 0),

    -- Timestamps
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at        TIMESTAMPTZ
);

-- Core query indexes
CREATE INDEX idx_listings_status        ON listings(status);
CREATE INDEX idx_listings_municipality  ON listings(municipality);
CREATE INDEX idx_listings_property_type ON listings(property_type);
CREATE INDEX idx_listings_price_mxn     ON listings(price_mxn);
CREATE INDEX idx_listings_owner_id      ON listings(owner_id);
CREATE INDEX idx_listings_is_featured   ON listings(is_featured);
CREATE INDEX idx_listings_published_at  ON listings(published_at DESC);

-- Full text search in Spanish (title + description + location)
ALTER TABLE listings
    ADD COLUMN search_vector TSVECTOR
    GENERATED ALWAYS AS (
        to_tsvector('spanish',
            coalesce(title, '')        || ' ' ||
            coalesce(description, '')  || ' ' ||
            coalesce(municipality, '') || ' ' ||
            coalesce(community, '')
        )
    ) STORED;

CREATE INDEX idx_listings_search ON listings USING GIN(search_vector);

CREATE TRIGGER set_listings_updated_at
    BEFORE UPDATE ON listings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
