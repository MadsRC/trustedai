#!/bin/sh

# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
#
# SPDX-License-Identifier: AGPL-3.0-only
#MISE description="Build the React frontend"
#MISE sources=["frontend/**/*"]
#MISE outputs={"auto"=true}

set -e

cd frontend
npm install
npm run build
cd ..

exit 0
