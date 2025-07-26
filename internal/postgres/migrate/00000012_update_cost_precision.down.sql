-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Rollback fractional cents migration

-- Add back the old integer cents columns with temp names
ALTER TABLE usage_events 
ADD COLUMN input_cost_cents_old BIGINT,
ADD COLUMN output_cost_cents_old BIGINT,
ADD COLUMN total_cost_cents_old BIGINT;

-- Migrate data back from fractional cents to integer cents (truncate)
UPDATE usage_events 
SET input_cost_cents_old = COALESCE(FLOOR(input_cost_cents), 0)::BIGINT,
    output_cost_cents_old = COALESCE(FLOOR(output_cost_cents), 0)::BIGINT,
    total_cost_cents_old = COALESCE(FLOOR(total_cost_cents), 0)::BIGINT
WHERE input_cost_cents IS NOT NULL OR output_cost_cents IS NOT NULL OR total_cost_cents IS NOT NULL;

-- Drop fractional cents columns
ALTER TABLE usage_events 
DROP COLUMN input_cost_cents,
DROP COLUMN output_cost_cents,
DROP COLUMN total_cost_cents;

-- Rename old columns back to original names
ALTER TABLE usage_events 
RENAME COLUMN input_cost_cents_old TO input_cost_cents;
ALTER TABLE usage_events 
RENAME COLUMN output_cost_cents_old TO output_cost_cents;
ALTER TABLE usage_events 
RENAME COLUMN total_cost_cents_old TO total_cost_cents;