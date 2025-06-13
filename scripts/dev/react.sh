#!/bin/sh

# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
#
# SPDX-License-Identifier: AGPL-3.0-only
#MISE description="Start the React development server"

set -e

cd frontend
npm run start
cd ..

exit 0