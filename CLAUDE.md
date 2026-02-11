# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

colony-relay is a per-project message relay for agent-to-agent communication. A single Go binary provides an HTTP server (SQLite-backed) and CLI tools for sending/receiving messages. Agents discover the server automatically via a port file written to `.colony-relay/`.

## Commands

```bash
# Run all tests
go test ./...

# Run a single test
go test -run TestStreamSSE ./pkg/relay/

# Build the binary
go build -o bin/colony-relay ./cmd/colony-relay/

# Install globally
go install ./cmd/colony-relay/

# Manual smoke test
colony-relay init
colony-relay start &
colony-relay say --from alice "hello @bob"
colony-relay hear --for bob
colony-relay status
```

## Architecture

Three packages, one binary:

- **`pkg/relay`** - Core server. `Store` (SQLite via modernc.org/sqlite, no CGO) handles persistence. `Server` implements HTTP handlers and SSE broadcasting. `ParseMentions` extracts @names from message bodies. Web UI is embedded via `//go:embed`.

- **`pkg/discover`** - Server discovery. Walks up the directory tree from CWD looking for `.colony-relay/port`. Falls back to `--server` flag, then `RELAY_SERVER` env var.

- **`pkg/skill`** - Embeds the Claude Code skill template (`relay.md`) that `init` installs to `.claude/commands/relay.md`.

- **`cmd/colony-relay`** - Subcommand dispatch. Each subcommand is a separate file (`start.go`, `say.go`, `hear.go`, `init.go`, `status.go`). Uses stdlib `flag` only, no CLI framework.

## Key behaviors

- **Auto port increment**: `start` tries port 4100, increments up to 4199 if busy. The actual port is written to `.colony-relay/port`.
- **Graceful shutdown**: `start` removes port/pid files on SIGINT/SIGTERM.
- **Presence**: Updated when an agent POSTs a message or GETs messages with `?for=`. Default 30-minute window.
- **Message filtering**: `GetForEntity` uses SQL `LIKE` on the message body (case-insensitive), not the stored mentions JSON. This means bare names without `@` also match.
- **Polling state**: `hear` tracks last-seen message ID in `.colony-relay/<name>.lastid` so repeated calls return only new messages.

## Testing

Tests use in-memory SQLite (`:memory:`), `httptest`, and real `net.Listen` for port tests. No mocking. Table-driven tests in `parser_test.go`.
