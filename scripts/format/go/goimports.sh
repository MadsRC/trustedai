#!/bin/sh

# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
#
# SPDX-License-Identifier: AGPL-3.0-only

set -e

# Check if goimports is installed
if ! command -v goimports > /dev/null 2>&1; then
    echo "Error: goimports is not installed. Install with: go install golang.org/x/tools/cmd/goimports@latest"
    exit 1
fi

# Format mode - fix formatting issues silently unless there's an error
# Run goimports on each file, capturing any errors
# Use find with -exec to avoid subshell issues
find . -name "*.go" -not -path "./vendor/*" -not -path "./gen/*" -not -path "*/\.*" -exec sh -c '
    if ! goimports -e -w "$1" 2>/dev/null; then
        echo "Error formatting file: $1"
        exit 1
    fi
' sh {} \;
# No output on success

exit 0
