-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Consolidated database migration - creates complete schema

-- Create organizations table first (foundation for other relations)
CREATE TABLE IF NOT EXISTS organizations (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    is_system BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    sso_type VARCHAR(50) NOT NULL,
    sso_config JSONB NOT NULL
);

-- Create users table with organization relationship
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    external_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    system_admin BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_login TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT valid_provider CHECK (provider IN ('oidc', 'saml', 'github', 'none'))
);

-- Create API tokens table for both human and machine users
CREATE TABLE IF NOT EXISTS tokens (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    description TEXT,
    prefix_hash VARCHAR(255) NOT NULL UNIQUE,
    token_hash TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE
);

-- Create credential tables for different provider types
CREATE TABLE openrouter_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    api_key VARCHAR(1024) NOT NULL,
    site_name VARCHAR(255),
    http_referer VARCHAR(512),
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create models table with metadata instead of model_reference
CREATE TABLE models (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    credential_id UUID,
    credential_type VARCHAR(50) NOT NULL,
    pricing JSONB NOT NULL DEFAULT '{}',
    capabilities JSONB NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraint to ensure credential_id and credential_type match
    CONSTRAINT valid_credential_reference CHECK (
        (credential_type = 'openrouter' AND credential_id IS NOT NULL) OR
        (credential_type != 'openrouter' AND credential_id IS NOT NULL)
    )
);

-- Create indexes for all tables
CREATE INDEX idx_organizations_name ON organizations(name);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_organization_id ON users(organization_id);
CREATE UNIQUE INDEX idx_users_provider_external_id ON users(provider, external_id);
CREATE INDEX idx_tokens_prefix_hash ON tokens(prefix_hash);
CREATE INDEX idx_tokens_user_id ON tokens(user_id);
CREATE INDEX idx_tokens_expires_at ON tokens(expires_at);
CREATE INDEX idx_models_provider_id ON models(provider_id);
CREATE INDEX idx_models_enabled ON models(enabled);
CREATE INDEX idx_models_credential_type ON models(credential_type);
CREATE INDEX idx_models_credential_id ON models(credential_id);
CREATE INDEX idx_models_metadata ON models USING gin(metadata);
CREATE INDEX idx_openrouter_credentials_enabled ON openrouter_credentials(enabled);

-- Add table and column comments
COMMENT ON TABLE tokens IS 'Stores API tokens for both human and machine users';
COMMENT ON COLUMN tokens.prefix_hash IS 'Hash of token prefix for fast lookups';
COMMENT ON COLUMN tokens.token_hash IS 'Argon2id hash of full token with parameters';
COMMENT ON COLUMN users.provider IS 'Authentication provider (oidc, saml, github, none)';

-- Create a function to validate credential references
CREATE OR REPLACE FUNCTION validate_credential_reference() 
RETURNS TRIGGER AS $$
BEGIN
    -- Check if the credential exists based on type
    IF NEW.credential_type = 'openrouter' THEN
        IF NOT EXISTS (SELECT 1 FROM openrouter_credentials WHERE id = NEW.credential_id AND enabled = true) THEN
            RAISE EXCEPTION 'Referenced OpenRouter credential does not exist or is disabled';
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to validate credential references
CREATE TRIGGER validate_model_credential_reference
    BEFORE INSERT OR UPDATE ON models
    FOR EACH ROW
    EXECUTE FUNCTION validate_credential_reference();
