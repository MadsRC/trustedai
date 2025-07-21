-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Add cost tracking columns to usage_events table
ALTER TABLE usage_events 
ADD COLUMN input_cost_cents BIGINT,
ADD COLUMN output_cost_cents BIGINT,
ADD COLUMN total_cost_cents BIGINT;