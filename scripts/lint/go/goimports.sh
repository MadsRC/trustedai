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

# Verify mode - check formatting without modifying files
# Only output if there are issues
# Use find with -exec to avoid subshell issues
UNFORMATTED=$(find . -name "*.go" -not -path "./vendor/*" -not -path "*/\.*" -exec sh -c 'if goimports -e -d -l "$1" | grep -q .; then echo "$1"; fi' sh {} \;)

if [ -n "$UNFORMATTED" ]; then
    echo "The following files are not formatted correctly:"
    echo "$UNFORMATTED"
    echo "Run 'mise run format:go:goimports' to fix formatting issues"
    exit 1
fi
# No output on success

exit 0
