-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

CREATE INDEX CONCURRENTLY idx_usage_data_complete ON usage_events(data_complete, timestamp) WHERE data_complete = false;