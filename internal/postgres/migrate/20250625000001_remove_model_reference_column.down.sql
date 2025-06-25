-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Migration rollback: restore model_reference column and remove metadata column

-- Add back the model_reference column
ALTER TABLE models ADD COLUMN model_reference VARCHAR(512) NOT NULL DEFAULT '';

-- Migrate data back from metadata to model_reference
UPDATE models 
SET model_reference = COALESCE(metadata->>'model_reference', '')
WHERE metadata ? 'model_reference';

-- Create index on model_reference
CREATE INDEX IF NOT EXISTS idx_models_model_reference ON models(model_reference);

-- Drop the metadata column and its index
DROP INDEX IF EXISTS idx_models_metadata;
ALTER TABLE models DROP COLUMN IF EXISTS metadata;