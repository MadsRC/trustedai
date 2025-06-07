-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Create API tokens table for both human and machine users
CREATE TABLE IF NOT EXISTS tokens (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    description TEXT,
    prefix_hash VARCHAR(255) NOT NULL UNIQUE, -- SHA3-256 of first 8 chars (base64 encoded)
    token_hash TEXT NOT NULL, -- Argon2id hash string
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE
);

COMMENT ON TABLE tokens IS 'Stores API tokens for both human and machine users';
COMMENT ON COLUMN tokens.prefix_hash IS 'Hash of token prefix for fast lookups';
COMMENT ON COLUMN tokens.token_hash IS 'Argon2id hash of full token with parameters';

-- Add indexes for common access patterns
CREATE INDEX idx_tokens_prefix_hash ON tokens(prefix_hash);
CREATE INDEX idx_tokens_user_id ON tokens(user_id);
CREATE INDEX idx_tokens_expires_at ON tokens(expires_at);
