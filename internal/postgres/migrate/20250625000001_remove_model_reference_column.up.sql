-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Migration to add metadata column and remove model_reference column
-- since model_reference is now stored in gai.Model.Metadata

-- First add the metadata column if it doesn't exist
ALTER TABLE models ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';

-- Migrate existing model_reference data to metadata column
UPDATE models 
SET metadata = jsonb_set(
    COALESCE(metadata, '{}'),
    '{model_reference}',
    to_jsonb(model_reference)
)
WHERE model_reference IS NOT NULL AND model_reference != '';

-- Remove the model_reference column
ALTER TABLE models DROP COLUMN IF EXISTS model_reference;

-- Drop the index on model_reference if it exists
DROP INDEX IF EXISTS idx_models_model_reference;

-- Create index on metadata for performance
CREATE INDEX IF NOT EXISTS idx_models_metadata ON models USING gin(metadata);