-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Remove cost tracking columns from usage_events table
ALTER TABLE usage_events 
DROP COLUMN IF EXISTS input_cost_cents,
DROP COLUMN IF EXISTS output_cost_cents,
DROP COLUMN IF EXISTS total_cost_cents;