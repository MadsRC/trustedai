#!/bin/sh
# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
# SPDX-License-Identifier: AGPL-3.0-only
#MISE description="Run unit tests for the frontend"
#MISE depends=["build:react"]

# Run frontend unit tests
cd frontend && npm run test:unit
