#!/bin/sh

# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
#
# SPDX-License-Identifier: AGPL-3.0-only
#MISE sources=["**/*.sh", "**/_default"]
#MISE outputs={"auto"=true}

# Check if shellcheck is installed
if ! command -v shellcheck > /dev/null 2>&1; then
    echo "Error: shellcheck is not installed"
    exit 1
fi

# Find all shell scripts with .sh extension
find . -type f -name "*.sh" -not -path "*/\.*" -not -path "*/vendor/*" -not -path "*/node_modules/*" -not -path "*/build/*" | sort | while read -r file; do
    # Run shellcheck silently first
    if ! shellcheck -q "$file" > /dev/null 2>&1; then
        # If the silent run fails, run again with output
        shellcheck "$file"
        # Don't exit here, continue checking other files
    fi
done

# Find files with shell shebang
find . -type f -not -path "*/\.*" -not -path "*/vendor/*" -not -path "*/node_modules/*" -not -path "*/build/*" | sort | while read -r file; do
    if [ -f "$file" ] && head -n 1 "$file" | grep -q '^#!/bin/\(bash\|sh\|zsh\)' && echo "$file" | grep -qv '\.sh$'; then
        # Run shellcheck silently first
        if ! shellcheck -q "$file" > /dev/null 2>&1; then
            # If the silent run fails, run again with output
            shellcheck "$file"
            # Don't exit here, continue checking other files
        fi
    fi
done

# Always exit with success to avoid breaking the mise run chain
exit 0
