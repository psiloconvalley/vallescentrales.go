-- Migration 0002: Wizard Onboarding, Property Listings, and Monetization Tables
-- Safe and idempotent for all table states

BEGIN;

-- 1. Ensure user columns for onboarding exist
ALTER TABLE users ADD COLUMN IF NOT EXISTS user_type VARCHAR(50) DEFAULT 'buyer';
ALTER TABLE users ADD COLUMN IF NOT EXISTS onboarding_completed BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS municipality VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS show_phone BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS show_whatsapp BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS bio TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS agency_name VARCHAR(150);

CREATE INDEX IF NOT EXISTS idx_users_onboarding ON users(onboarding_completed);
CREATE INDEX IF NOT EXISTS idx_users_type ON users(user_type);

-- 2. Property Listings Table (for wizard & bienes raíces)
CREATE TABLE IF NOT EXISTS properties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    property_type VARCHAR(50) NOT NULL,
    operation_type VARCHAR(20) NOT NULL,
    legal_regime VARCHAR(50) NOT NULL DEFAULT 'escritura_publica',
    price NUMERIC(14,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'MXN',
    price_negotiable BOOLEAN NOT NULL DEFAULT FALSE,
    built_area_m2 NUMERIC(10,2),
    total_area_m2 NUMERIC(10,2) NOT NULL,
    bedrooms INT DEFAULT 0,
    bathrooms NUMERIC(3,1) DEFAULT 0,
    half_bathrooms INT DEFAULT 0,
    parking_spaces INT DEFAULT 0,
    floors INT DEFAULT 1,
    year_built INT,
    amenities TEXT[] DEFAULT '{}',
    services TEXT[] DEFAULT '{}',
    municipality VARCHAR(100) NOT NULL,
    neighborhood VARCHAR(150),
    address VARCHAR(255),
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    status VARCHAR(20) NOT NULL DEFAULT 'published',
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,
    featured_until TIMESTAMPTZ,
    views_count INT NOT NULL DEFAULT 0,
    favorites_count INT NOT NULL DEFAULT 0,
    contact_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_properties_user ON properties(user_id);
CREATE INDEX IF NOT EXISTS idx_properties_municipality ON properties(municipality);
CREATE INDEX IF NOT EXISTS idx_properties_op_type ON properties(operation_type, property_type);
CREATE INDEX IF NOT EXISTS idx_properties_status ON properties(status);
CREATE INDEX IF NOT EXISTS idx_properties_featured ON properties(is_featured) WHERE is_featured = TRUE;
CREATE INDEX IF NOT EXISTS idx_properties_price ON properties(price);
CREATE INDEX IF NOT EXISTS idx_properties_created ON properties(created_at DESC);

-- 3. Property Images Table
CREATE TABLE IF NOT EXISTS property_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    storage_url TEXT NOT NULL,
    thumbnail_url TEXT,
    display_order INT NOT NULL DEFAULT 0,
    is_hero BOOLEAN NOT NULL DEFAULT FALSE,
    caption VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_property_images_property ON property_images(property_id);
CREATE INDEX IF NOT EXISTS idx_property_images_hero ON property_images(property_id, is_hero);

-- 4. Subscriptions & Tier Table
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier VARCHAR(50) NOT NULL,
    billing_cycle VARCHAR(20) NOT NULL DEFAULT 'monthly',
    status VARCHAR(30) NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_end TIMESTAMPTZ NOT NULL,
    payment_method VARCHAR(50),
    external_sub_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);

-- 5. Payments Table (safely alter existing table or create new)
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(10,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'MXN',
    payment_method VARCHAR(50) NOT NULL,
    concept VARCHAR(100) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    reference_code VARCHAR(100) UNIQUE,
    spei_clabe VARCHAR(18),
    oxxo_barcode TEXT,
    stripe_payment_id VARCHAR(255),
    property_id UUID REFERENCES properties(id) ON DELETE SET NULL,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    paid_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ensure all payment columns exist even if payments table was already created in 0001
ALTER TABLE payments ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS amount NUMERIC(10,2);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS currency VARCHAR(3) DEFAULT 'MXN';
ALTER TABLE payments ADD COLUMN IF NOT EXISTS payment_method VARCHAR(50);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS concept VARCHAR(100);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS status VARCHAR(30) DEFAULT 'pending';
ALTER TABLE payments ADD COLUMN IF NOT EXISTS reference_code VARCHAR(100);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS spei_clabe VARCHAR(18);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS oxxo_barcode TEXT;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS stripe_payment_id VARCHAR(255);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS property_id UUID REFERENCES properties(id) ON DELETE SET NULL;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS notes TEXT;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_payments_user ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_ref ON payments(reference_code);

-- 6. Property Leads / Inquiries
CREATE TABLE IF NOT EXISTS property_leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    buyer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    buyer_name VARCHAR(150) NOT NULL,
    buyer_email VARCHAR(255) NOT NULL,
    buyer_phone VARCHAR(50),
    buyer_whatsapp VARCHAR(50),
    message TEXT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'new',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_leads_property ON property_leads(property_id);
CREATE INDEX IF NOT EXISTS idx_leads_buyer ON property_leads(buyer_id);

-- 7. Platform Ads
CREATE TABLE IF NOT EXISTS advertisements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(200) NOT NULL,
    client_name VARCHAR(200) NOT NULL,
    client_contact VARCHAR(200),
    target_zone VARCHAR(100),
    target_vertical VARCHAR(50) DEFAULT 'all',
    banner_url TEXT NOT NULL,
    destination_url TEXT NOT NULL,
    placement VARCHAR(50) NOT NULL DEFAULT 'sidebar',
    impressions_bought INT NOT NULL DEFAULT 0,
    impressions_served INT NOT NULL DEFAULT 0,
    clicks INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ads_active ON advertisements(is_active, starts_at, ends_at);
CREATE INDEX IF NOT EXISTS idx_ads_vertical ON advertisements(target_vertical);

COMMIT;
