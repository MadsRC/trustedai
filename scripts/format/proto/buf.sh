#!/bin/sh

# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
#
# SPDX-License-Identifier: AGPL-3.0-only
#MISE sources=["**/*.proto"]
#MISE outputs={"auto"=true}

set -e

# Check if buf is installed
if ! command -v buf > /dev/null 2>&1; then
    echo "Error: buf is not installed."
    exit 1
fi

buf format -w

exit 0
