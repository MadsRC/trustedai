#!/bin/sh

# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
#
# SPDX-License-Identifier: AGPL-3.0-only
#MISE sources=["**/*.go"]
#MISE outputs={"auto"=true}

set -e

# Check if golangci-lint is installed
if ! command -v golangci-lint > /dev/null 2>&1; then
    echo "Error: golangci-lint is not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
    exit 1
fi

# Run golangci-lint and capture its output
output=$(golangci-lint run ./... 2>&1) || {
  # If the command failed, output the captured output
  echo "$output"
  exit 1
}

# If we reach here, the command succeeded with no findings
exit 0
