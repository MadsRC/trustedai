-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

BEGIN;

-- Instead of using procedural code, directly attempt to drop the constraint
-- This is more reliable across PostgreSQL versions
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_provider_check;

-- Add new check constraint with updated values
ALTER TABLE users
ADD CONSTRAINT valid_provider
CHECK (provider IN ('oidc', 'saml', 'github', 'none'));

COMMENT ON COLUMN users.provider IS 'Authentication provider (oidc, saml, github, none)';

COMMIT;
