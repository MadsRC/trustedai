#!/bin/sh

# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
#
# SPDX-License-Identifier: AGPL-3.0-only

# Run reuse lint with quiet flag first
if ! reuse lint -q; then
    # If the quiet run fails, run without quiet flag to show details
    reuse lint
    exit 1
fi

exit 0
