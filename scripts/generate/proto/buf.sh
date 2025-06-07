#!/bin/sh

# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
#
# SPDX-License-Identifier: AGPL-3.0-only
#MISE description="Generate protobuf files with buf"
#MISE sources=["**/*.proto"]
#MISE outputs={"auto"=true}

set -e

# Check if buf is installed
if ! command -v buf > /dev/null 2>&1; then
    echo "Error: buf is not installed."
    exit 1
fi

# Run buf and capture its output
output=$(buf generate 2>&1) || {
  # If the command failed, output the captured output
  echo "$output"
  exit 1
}

# If we reach here, the command succeeded with no findings
exit 0
