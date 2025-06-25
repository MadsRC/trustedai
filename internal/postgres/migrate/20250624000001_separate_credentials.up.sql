-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Drop the existing tables to recreate with new structure
DROP TABLE IF EXISTS models;
DROP TABLE IF EXISTS providers;

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

-- Create models table without provider foreign key (providers are now hardcoded)
CREATE TABLE models (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    provider_id VARCHAR(255) NOT NULL, -- References hardcoded provider IDs like 'openrouter'
    credential_id UUID, -- Generic reference to any credential type
    credential_type VARCHAR(50) NOT NULL, -- 'openrouter', 'openai', etc.
    model_reference VARCHAR(512) NOT NULL, -- Composite reference like "openrouter:deepseek/deepseek-r1-0528-qwen3-8b:free"
    pricing JSONB NOT NULL DEFAULT '{}',
    capabilities JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraint to ensure credential_id and credential_type match
    CONSTRAINT valid_credential_reference CHECK (
        (credential_type = 'openrouter' AND credential_id IS NOT NULL) OR
        (credential_type != 'openrouter' AND credential_id IS NOT NULL)
    )
);

-- Create indexes
CREATE INDEX idx_models_provider_id ON models(provider_id);
CREATE INDEX idx_models_enabled ON models(enabled);
CREATE INDEX idx_models_credential_type ON models(credential_type);
CREATE INDEX idx_models_credential_id ON models(credential_id);
CREATE INDEX idx_models_model_reference ON models(model_reference);
CREATE INDEX idx_openrouter_credentials_enabled ON openrouter_credentials(enabled);

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