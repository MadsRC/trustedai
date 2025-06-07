-- SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS kubernetes_connectors;
DROP TABLE IF EXISTS connectors;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;
