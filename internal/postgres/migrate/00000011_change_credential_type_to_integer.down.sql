-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Rollback credential_type from INTEGER back to VARCHAR

-- Drop the numeric constraint
ALTER TABLE models DROP CONSTRAINT IF EXISTS valid_credential_reference;

-- Change the column type back to VARCHAR
ALTER TABLE models ALTER COLUMN credential_type TYPE VARCHAR(50) USING credential_type::VARCHAR;

-- Convert numeric values back to string equivalents
UPDATE models SET credential_type = 'openrouter' WHERE credential_type = '1';
UPDATE models SET credential_type = 'unspecified' WHERE credential_type = '0';

-- Add the original string constraint
ALTER TABLE models ADD CONSTRAINT valid_credential_reference CHECK (
    (credential_type = 'openrouter' AND credential_id IS NOT NULL) OR
    (credential_type != 'openrouter' AND credential_id IS NOT NULL)
);

-- Restore the original trigger function
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