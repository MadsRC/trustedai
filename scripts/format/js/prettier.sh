#!/bin/sh
# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
# SPDX-License-Identifier: AGPL-3.0-only
#MISE description="Format JavaScript/TypeScript files with prettier"

set -e

# Format files with prettier
cd frontend && npx prettier --write --log-level warn "**/*.{js,jsx,ts,tsx,json,css,md}"

exit 0