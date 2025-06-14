#!/bin/sh

# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
#
# SPDX-License-Identifier: AGPL-3.0-only
#MISE sources=["**/*.go"]
#MISE outputs={"auto"=true}

set -e

# Run the modernize tool and capture its output
# We use ./... which is a Go-specific pattern that modernize understands
output=$(go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -test ./... 2>&1) || {
  # If the command failed, output the captured output
  echo "$output"
  exit 1
}

# If we reach here, the command succeeded
exit 0
