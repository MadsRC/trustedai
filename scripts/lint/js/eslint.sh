#!/bin/sh
# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
# SPDX-License-Identifier: AGPL-3.0-only
#MISE description="Check JavaScript/TypeScript files with eslint"
#MISE sources=["frontend/**/*"]
#MISE outputs={"auto"=true}

set -e

# Run eslint on JavaScript/TypeScript files
cd frontend && eslint "src/**/*.{js,jsx,ts,tsx}"

exit 0
