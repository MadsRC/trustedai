-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Update cost columns to use NUMERIC(20,6) for fractional cent precision
-- This allows storing fractional cents like 0.003800 cents exactly

-- Add new fractional cents columns with temp names
ALTER TABLE usage_events 
ADD COLUMN input_cost_cents_new NUMERIC(20,6),
ADD COLUMN output_cost_cents_new NUMERIC(20,6),
ADD COLUMN total_cost_cents_new NUMERIC(20,6);

-- Migrate existing data from integer cents to fractional cents
UPDATE usage_events 
SET input_cost_cents_new = COALESCE(input_cost_cents, 0)::NUMERIC(20,6),
    output_cost_cents_new = COALESCE(output_cost_cents, 0)::NUMERIC(20,6),
    total_cost_cents_new = COALESCE(total_cost_cents, 0)::NUMERIC(20,6)
WHERE input_cost_cents IS NOT NULL OR output_cost_cents IS NOT NULL OR total_cost_cents IS NOT NULL;

-- Drop old integer cents columns
ALTER TABLE usage_events 
DROP COLUMN input_cost_cents,
DROP COLUMN output_cost_cents,
DROP COLUMN total_cost_cents;

-- Rename new columns to original names
ALTER TABLE usage_events 
RENAME COLUMN input_cost_cents_new TO input_cost_cents;
ALTER TABLE usage_events 
RENAME COLUMN output_cost_cents_new TO output_cost_cents;
ALTER TABLE usage_events 
RENAME COLUMN total_cost_cents_new TO total_cost_cents;