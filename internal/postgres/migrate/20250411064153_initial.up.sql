-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Create organizations table first (foundation for other relations)
CREATE TABLE IF NOT EXISTS organizations (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    is_system BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    sso_type VARCHAR(50) NOT NULL,  -- Changed to NOT NULL
    sso_config JSONB NOT NULL       -- Removed default, config is required
);

-- Create users table with organization relationship
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    external_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('oidc', 'saml', 'github', 'google')),  -- Added enum constraint
    system_admin BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_login TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Original connectors tables (remain unchanged)
CREATE TABLE IF NOT EXISTS connectors (
    id VARCHAR(255) PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    last_seen BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS kubernetes_connectors (
    connector_id VARCHAR(255) PRIMARY KEY,
    version VARCHAR(50) NOT NULL,
    cluster_id VARCHAR(255) NOT NULL,
    api_endpoint VARCHAR(255) NOT NULL,
    FOREIGN KEY (connector_id) REFERENCES connectors(id) ON DELETE CASCADE
);

-- Add indexes for all tables
CREATE INDEX idx_organizations_name ON organizations(name);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_organization_id ON users(organization_id);
CREATE UNIQUE INDEX idx_users_provider_external_id ON users(provider, external_id);
CREATE INDEX idx_connectors_type ON connectors(type);
CREATE INDEX idx_connectors_status ON connectors(status);
