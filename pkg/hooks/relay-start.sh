#!/bin/bash
# ABOUTME: SessionStart hook that announces presence on the relay.
# ABOUTME: Catches up on recent messages and injects them as context.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=relay-resolve.sh
source "$SCRIPT_DIR/relay-resolve.sh"

INPUT="$(cat)"
BIN="$(resolve_relay_bin "$INPUT")" || exit 0
NAME="$(resolve_relay_name "$INPUT")"

# Announce presence
"$BIN" say --from "$NAME" "online" 2>/dev/null || exit 0

# Catch up on recent messages
MESSAGES="$("$BIN" hear --for "$NAME" --all --limit 5 2>/dev/null)" || exit 0

if [ -n "$MESSAGES" ]; then
  echo "$MESSAGES" | jq -Rs '{additionalContext: ("Recent relay messages:\n" + .)}'
fi

exit 0
