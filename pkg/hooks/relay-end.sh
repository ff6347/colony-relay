#!/bin/bash
# ABOUTME: SessionEnd hook that announces the agent is going offline.
# ABOUTME: Runs async so it does not block session teardown.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=relay-resolve.sh
source "$SCRIPT_DIR/relay-resolve.sh"

INPUT="$(cat)"
BIN="$(resolve_relay_bin "$INPUT")" || exit 0
NAME="$(resolve_relay_name "$INPUT")"

"$BIN" say --from "$NAME" "going offline" 2>/dev/null || true

exit 0
