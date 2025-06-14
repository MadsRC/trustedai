#!/bin/sh
# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
# SPDX-License-Identifier: AGPL-3.0-only
#MISE description="Check if JavaScript/TypeScript files are formatted with prettier"
#MISE sources=["**/*.go"]
#MISE outputs={"auto"=true}

set -e

# Check if files are formatted with prettier
cd frontend && prettier --check "**/*.{js,jsx,ts,tsx,json,css,md}"

exit 0
