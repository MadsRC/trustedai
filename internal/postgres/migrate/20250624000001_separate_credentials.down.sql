-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Drop trigger and function
DROP TRIGGER IF EXISTS validate_model_credential_reference ON models;
DROP FUNCTION IF EXISTS validate_credential_reference();

-- Drop tables
DROP TABLE IF EXISTS models;
DROP TABLE IF EXISTS openrouter_credentials;