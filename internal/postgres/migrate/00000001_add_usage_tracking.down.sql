-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Drop tables (billing_summaries first due to no foreign key dependencies)
DROP TABLE IF EXISTS billing_summaries;
DROP TABLE IF EXISTS usage_events;