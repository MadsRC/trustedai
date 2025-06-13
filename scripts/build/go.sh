#!/bin/sh

# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
#
# SPDX-License-Identifier: AGPL-3.0-only
#MISE description="Build the Go project"
#MISE sources=["go.mod", "go.sum", "**/*.go"]
#MISE outputs={"auto"=true}

set -e

# Get the project root directory (parent of llmgw directory)
PROJECT_ROOT=$(git rev-parse --show-toplevel)

# Define build directory
BUILD_DIR="$PROJECT_ROOT/build"

# Create build directory if it doesn't exist
mkdir -p "$BUILD_DIR" > /dev/null 2>&1

# Build variables
BINARY_NAME="llmgw"

# Determine OS and architecture
GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

# Set output filename with OS and architecture
OUTPUT_FILENAME="${BINARY_NAME}_${GOOS}_${GOARCH}"

# Create a temporary file for build output
BUILD_LOG=$(mktemp)

# Build the application from the current directory
echo "Building $OUTPUT_FILENAME..."
if ! go build -o "$BUILD_DIR/$OUTPUT_FILENAME" ./cmd/llmgw > "$BUILD_LOG" 2>&1; then
    echo "Error: Failed to build $OUTPUT_FILENAME"
    echo "--- Build errors ---"
    cat "$BUILD_LOG"
    echo "-------------------"
    rm -f "$BUILD_LOG"
    exit 1
fi

# Clean up the log file
rm -f "$BUILD_LOG"

# Only show output if the binary doesn't exist or has zero size
if [ ! -s "$BUILD_DIR/$OUTPUT_FILENAME" ]; then
    echo "Error: Build failed, binary not created or has zero size"
    exit 1
fi

echo "Successfully built $OUTPUT_FILENAME"
exit 0
