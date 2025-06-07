-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

BEGIN;

-- Remove new constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS valid_provider;

-- Restore original check constraint
ALTER TABLE users
ADD CONSTRAINT users_provider_check
CHECK (provider IN ('oidc', 'saml', 'github', 'google'));

COMMENT ON COLUMN users.provider IS 'Authentication provider (oidc, saml, github, google)';

COMMIT;
