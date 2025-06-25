-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Consolidated database rollback - drops all tables and objects

-- Drop trigger and function
DROP TRIGGER IF EXISTS validate_model_credential_reference ON models;
DROP FUNCTION IF EXISTS validate_credential_reference();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS models;
DROP TABLE IF EXISTS openrouter_credentials;
DROP TABLE IF EXISTS kubernetes_connectors;
DROP TABLE IF EXISTS connectors;
DROP TABLE IF EXISTS tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;