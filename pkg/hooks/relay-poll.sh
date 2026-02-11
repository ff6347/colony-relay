#!/bin/bash
# ABOUTME: UserPromptSubmit hook that polls the relay for new messages.
# ABOUTME: Injects any new messages as context before each turn.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=relay-resolve.sh
source "$SCRIPT_DIR/relay-resolve.sh"

INPUT="$(cat)"
BIN="$(resolve_relay_bin "$INPUT")" || exit 0
NAME="$(resolve_relay_name "$INPUT")"

MESSAGES="$("$BIN" hear --for "$NAME" --limit 5 2>/dev/null)" || exit 0

if [ -n "$MESSAGES" ]; then
  echo "$MESSAGES" | jq -Rs '{additionalContext: ("Relay messages for '"$NAME"':\n" + .)}'
fi

exit 0
