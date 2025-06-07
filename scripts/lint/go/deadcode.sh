#!/bin/sh

# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
#
# SPDX-License-Identifier: AGPL-3.0-only

set -e

# Run the deadcode tool and capture its output
# We use ./... which is a Go-specific pattern that deadcode understands
output=$(go run golang.org/x/tools/cmd/deadcode@latest ./... 2>&1)

# Check if there was any output (findings)
if [ -n "$output" ]; then
  echo "$output"
  exit 1
fi

# If we reach here, the command succeeded with no findings
exit 0
