#!/bin/bash

# SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
#
# SPDX-License-Identifier: AGPL-3.0-only
#MISE description="Sets up or updates the cached magentic-ui repository for demo generation"
set -e

REPO_URL="https://github.com/microsoft/magentic-ui.git"
COMMIT_HASH="265708ed7d19d0c1904afef34ae1f428d00aa1a1"

# Determine project root (assuming script is in project_root/scripts/dev/)
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
PROJECT_ROOT=$(cd "$SCRIPT_DIR/../.." && pwd)

BUILD_DIR="$PROJECT_ROOT/build"
CACHE_REPO_DIR="$BUILD_DIR/magentic_ui_cache"

# Ensure build directory exists
mkdir -p "$BUILD_DIR"

# Function to prepare the cached repository
prepare_cached_repo() {
  local CACHE_ACTION_TAKEN=false

  # Check 1: Existence and initial clone
  if [ ! -d "$CACHE_REPO_DIR/.git" ]; then
    echo "Magentic UI cache not found. Cloning $REPO_URL to $CACHE_REPO_DIR..."
    rm -rf "$CACHE_REPO_DIR" # Ensure clean slate
    git clone --quiet "$REPO_URL" "$CACHE_REPO_DIR"
    (cd "$CACHE_REPO_DIR" && git checkout --quiet "$COMMIT_HASH" --force)
    echo "Successfully cloned and checked out commit $COMMIT_HASH."
    CACHE_ACTION_TAKEN=true
  else
    # Cache exists, perform checks
    local RECLONE_NEEDED=false

    # Check 2: Remote URL
    if ! (cd "$CACHE_REPO_DIR" && [ "$(git config --get remote.origin.url 2>/dev/null)" = "$REPO_URL" ]); then
      echo "Magentic UI cache has incorrect remote URL. Re-cloning..."
      RECLONE_NEEDED=true
    fi

    if [ "$RECLONE_NEEDED" = "false" ]; then
      # Fetch latest changes from remote quietly, ignore errors for fetch as repo might be offline
      (cd "$CACHE_REPO_DIR" && git fetch --quiet origin) 2>/dev/null || echo "Warning: 'git fetch' failed, proceeding with local cache check."

      # Check 3: Current commit. Get full commit hash for robust comparison.
      # Resolve target COMMIT_HASH to its full SHA within the context of the CACHE_REPO_DIR
      # Suppress errors if COMMIT_HASH is not found yet (e.g. after a failed fetch on a new repo)
      TARGET_FULL_COMMIT_HASH=$(cd "$CACHE_REPO_DIR" && git rev-parse "$COMMIT_HASH^{commit}" 2>/dev/null || echo "")
      CURRENT_FULL_COMMIT_HASH=$(cd "$CACHE_REPO_DIR" && git rev-parse HEAD 2>/dev/null || echo "")

      if [ -z "$TARGET_FULL_COMMIT_HASH" ]; then
          # This can happen if COMMIT_HASH is not in the fetched history yet.
          # A re-clone might be too aggressive if fetch failed due to network, 
          # but for script simplicity, if we can't resolve the target commit, we might need to re-evaluate.
          # For now, if target commit is unresolvable, assume something is wrong and potentially re-clone later if other checks imply it.
          # Or, if we are sure COMMIT_HASH should exist, this could be an error state leading to re-clone.
          echo "Warning: Target commit $COMMIT_HASH could not be resolved in the cache. If issues persist, a re-clone might be needed."
          # If we want to be strict and re-clone if target commit isn't found after fetch:
          # echo "Error: Target commit $COMMIT_HASH not found in repository. Re-cloning..."
          # RECLONE_NEEDED=true
      elif [ "$CURRENT_FULL_COMMIT_HASH" != "$TARGET_FULL_COMMIT_HASH" ]; then
        echo "Magentic UI cache is not at target commit $COMMIT_HASH. Checking out..."
        (cd "$CACHE_REPO_DIR" && git checkout --quiet "$COMMIT_HASH" --force)
        # Re-verify after checkout
        CURRENT_FULL_COMMIT_HASH_AFTER_CHECKOUT=$(cd "$CACHE_REPO_DIR" && git rev-parse HEAD 2>/dev/null || echo "")
        if [ "$CURRENT_FULL_COMMIT_HASH_AFTER_CHECKOUT" != "$TARGET_FULL_COMMIT_HASH" ]; then
            echo "Error: Failed to checkout target commit $COMMIT_HASH. Re-cloning..."
            RECLONE_NEEDED=true 
        else
            echo "Successfully checked out commit $COMMIT_HASH."
            CACHE_ACTION_TAKEN=true 
        fi
      fi
    fi
    
    # Check 4: Working directory cleanliness (only if not re-cloning)
    if [ "$RECLONE_NEEDED" = "false" ]; then
        if ! (cd "$CACHE_REPO_DIR" && test -z "$(git status --porcelain)"); then # if output is NOT empty, it's dirty
            echo "Magentic UI cache is dirty. Cleaning..."
            (cd "$CACHE_REPO_DIR" && git reset --hard HEAD && git clean -fdx)
            echo "Cached repository cleaned."
            CACHE_ACTION_TAKEN=true
        fi
    fi

    # If any check mandated a re-clone
    if [ "$RECLONE_NEEDED" = "true" ]; then
      rm -rf "$CACHE_REPO_DIR"
      git clone --quiet "$REPO_URL" "$CACHE_REPO_DIR"
      (cd "$CACHE_REPO_DIR" && git checkout --quiet "$COMMIT_HASH" --force)
      echo "Successfully re-cloned and checked out commit $COMMIT_HASH."
      CACHE_ACTION_TAKEN=true
    fi
  fi

  if [ "$CACHE_ACTION_TAKEN" = "true" ]; then
    echo "Magentic UI cache is ready at $CACHE_REPO_DIR."
  fi
  # If CACHE_ACTION_TAKEN is false, the script remains silent here.
  if [ "$CACHE_ACTION_TAKEN" = "true" ]; then
    return 0 # Action was taken
  else
    return 1 # No action was taken, cache was already fine
  fi
}

# Prepare the cached repo
if prepare_cached_repo; then # if function returned 0 (true, action taken)
  echo "-----------------------------------------------------"
  echo "Magentic UI cache setup process finished (actions were taken)."
else # if function returned 1 (false, no action taken, was already fine)
  # Script remains silent as requested
  :
fi
