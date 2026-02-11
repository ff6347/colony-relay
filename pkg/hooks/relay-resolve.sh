# ABOUTME: Shared helpers for relay hook scripts.
# ABOUTME: Resolves relay binary path, agent name, and project root.

ADJECTIVES=(swift bright calm bold keen sharp steady clear quick warm)
NOUNS=(fox owl elm oak ray arc flux node reef vale)

resolve_relay_bin() {
  if [ -n "${RELAY_BIN:-}" ]; then
    echo "$RELAY_BIN"
    return 0
  fi

  if command -v colony-relay >/dev/null 2>&1; then
    echo "colony-relay"
    return 0
  fi

  local cwd
  cwd="$(echo "$1" | jq -r '.cwd // empty' 2>/dev/null)"
  if [ -n "$cwd" ] && [ -x "$cwd/bin/colony-relay" ]; then
    echo "$cwd/bin/colony-relay"
    return 0
  fi

  return 1
}

resolve_project_root() {
  echo "$1" | jq -r '.cwd // empty' 2>/dev/null
}

generate_name() {
  local adj_idx noun_idx
  adj_idx=$((RANDOM % ${#ADJECTIVES[@]}))
  noun_idx=$((RANDOM % ${#NOUNS[@]}))
  echo "${ADJECTIVES[$adj_idx]}-${NOUNS[$noun_idx]}"
}

resolve_relay_name() {
  local input="$1"

  if [ -n "${RELAY_NAME:-}" ]; then
    echo "$RELAY_NAME"
    return 0
  fi

  local session_id root names_dir name_file
  session_id="$(echo "$input" | jq -r '.session_id // empty' 2>/dev/null)"
  root="$(resolve_project_root "$input")"

  if [ -n "$session_id" ] && [ -n "$root" ]; then
    names_dir="$root/.colony-relay/names"
    name_file="$names_dir/$session_id"

    if [ -f "$name_file" ]; then
      cat "$name_file"
      return 0
    fi

    # Generate and persist a new name
    mkdir -p "$names_dir"
    local name
    name="$(generate_name)"
    echo "$name" > "$name_file"
    echo "$name"
    return 0
  fi

  echo "${USER:-agent}"
}
