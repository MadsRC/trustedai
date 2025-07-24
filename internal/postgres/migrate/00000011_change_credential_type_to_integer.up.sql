-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Change credential_type from VARCHAR to INTEGER to match protobuf enum values

-- First, convert existing string values to numeric equivalents
UPDATE models SET credential_type = '1' WHERE credential_type = 'openrouter';
UPDATE models SET credential_type = '0' WHERE credential_type = 'unspecified' OR credential_type = '';

-- Drop the existing constraint
ALTER TABLE models DROP CONSTRAINT IF EXISTS valid_credential_reference;

-- Change the column type to INTEGER
ALTER TABLE models ALTER COLUMN credential_type TYPE INTEGER USING credential_type::INTEGER;

-- Add the new constraint with numeric values
ALTER TABLE models ADD CONSTRAINT valid_credential_reference CHECK (
    (credential_type = 1 AND credential_id IS NOT NULL) OR  -- OpenRouter
    (credential_type != 1 AND credential_id IS NOT NULL)    -- Other types
);

-- Update the trigger function to use numeric values
CREATE OR REPLACE FUNCTION validate_credential_reference()
RETURNS TRIGGER AS $$
BEGIN
    -- Check if the credential exists based on type
    IF NEW.credential_type = 1 THEN  -- OpenRouter
        IF NOT EXISTS (SELECT 1 FROM openrouter_credentials WHERE id = NEW.credential_id AND enabled = true) THEN
            RAISE EXCEPTION 'Referenced OpenRouter credential does not exist or is disabled';
        END IF;
    -- Add other credential type validations here as they are implemented
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;