-- migrations/000010_add_wallet_support.up.sql
-- Web3 wallet addresses for users
-- Supports Ethereum (EVM) and Solana wallets
-- One primary wallet per network per user

CREATE TABLE user_wallets (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    network     TEXT NOT NULL,
    address     TEXT NOT NULL,
    label       TEXT,
    is_primary  BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, network, address)
);

CREATE INDEX idx_wallets_user_id ON user_wallets(user_id);
CREATE INDEX idx_wallets_network ON user_wallets(network);
CREATE INDEX idx_wallets_address ON user_wallets(address);
