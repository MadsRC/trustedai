#!/bin/sh
# SPDX-FileCopyrightText: 2025 Mads Robin Havmand <mads@mrhavmand.dk>
#
# SPDX-License-Identifier: AGPL-3.0-only

# MISE:description Format all .proto files using buf format and report files that need formatting.
# MISE:usage scripts/lint/proto/buf-format.sh

# This script finds all .proto files and runs 'buf format --exit-code' on them.
# 'buf format --exit-code' exits 0 if the file is already formatted,
# and 1 if the file needs formatting (it also prints the formatted content to stdout).
# If 'buf format' encounters an error (e.g., invalid file), it may also exit non-zero and print to stderr.

# This script will:
# 1. Suppress the formatted content (stdout of 'buf format').
# 2. Allow 'buf format' error messages (stderr of 'buf format') to pass through.
# 3. Collect names of files for which 'buf format' exits non-zero.
# 4. If any such files are found, print their names to stderr and exit 1.
# 5. Otherwise, exit 0.

# Collect all filenames that need formatting or caused errors.
# The output of this entire pipeline (the echo'd filenames) will be captured by 'failed_files_list'.
failed_files_list=$( \
  find . -type f -name '*.proto' -print | while IFS= read -r proto_file; do \
    # Run 'buf format --exit-code'. Its stdout (formatted content) is suppressed.
    # Its stderr (buf errors) is allowed to pass through to the main script's stderr.
    if ! buf format --exit-code "$proto_file" > /dev/null; then \
      # If 'buf format' exited non-zero, print the filename.
      # This output is captured by 'failed_files_list'.
      echo "$proto_file"; \
    fi; \
  done \
)

# After checking all files, if 'failed_files_list' is not empty,
# print the collected filenames to stderr and exit with 1.
if [ -n "$failed_files_list" ]; then
  # Add a blank line for separation if buf might have printed errors to stderr.
  echo "" >&2
  echo "The following .proto files need formatting or caused an error with 'buf format':" >&2
  # Print each filename on a new line.
  # Use a POSIX-compliant way to iterate over lines in the 'failed_files_list' string.
  echo "$failed_files_list" | while IFS= read -r line; do
    # Ensure we don't print empty lines if any were captured (should not happen with echo "$proto_file")
    if [ -n "$line" ]; then
        echo "  $line" >&2
    fi
  done
  exit 1
fi

exit 0
