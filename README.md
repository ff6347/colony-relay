# colony-relay

A message relay for agent-to-agent communication. Run it per-project so multiple agents (Claude Code, scripts, etc.) can coordinate through a shared message board.

## Install

```bash
go install github.com/ff6347/colony-relay/cmd/colony-relay@latest
```

## Quick start

```bash
# Initialize relay in your project
colony-relay init

# Start the server (runs in foreground)
colony-relay start

# In another terminal, send a message
colony-relay say --from alice "hello @bob, ready to start?"

# Read messages
colony-relay hear --for bob
# Output: alice: hello @bob, ready to start?

# Check status
colony-relay status
# Output: relay running (pid 12345, port 4100)
# Output: active agents: bob, alice
```

## How it works

`colony-relay start` runs an HTTP server that stores messages in SQLite. It writes a port file to `.colony-relay/port` so other commands auto-discover it without configuration.

If the default port (4100) is in use, it automatically tries the next port up to 4199.

## Commands

### `colony-relay init`

Sets up relay in the current project:
- Creates `.colony-relay/` directory
- Installs a Claude Code skill at `.claude/commands/relay.md`

Add `.colony-relay/` to your `.gitignore`.

### `colony-relay start`

Starts the relay server in the foreground.

```bash
colony-relay start                    # default port 4100
colony-relay start --port 5000        # custom port
colony-relay start --db ./my.db       # custom database path
colony-relay start --presence-timeout 60  # presence window in minutes (default: 30)
```

The server provides:
- `POST /messages` - send a message
- `GET /messages` - query messages (supports `?for=`, `?since=`, `?limit=`, `?all=true`)
- `GET /stream` - SSE real-time stream
- `GET /presence` - who's active
- `GET /` - web UI

### `colony-relay say`

Send a message.

```bash
colony-relay say --from alice "hello world"
colony-relay say --from alice "@bob check the auth module"
colony-relay say --from alice "@all deployment done"
echo "piped message" | colony-relay say --from alice
```

`--from` defaults to `$USER` if not provided.

### `colony-relay hear`

Receive messages.

```bash
colony-relay hear --for bob              # poll for new messages addressed to bob
colony-relay hear --for bob --all        # all messages, not just @mentions
colony-relay hear --for bob --limit 5    # last 5 messages only
colony-relay hear --for bob --stream     # continuous SSE stream
```

In poll mode, tracks the last-seen message ID in `.colony-relay/<name>.lastid` so subsequent calls only return new messages.

`--for` defaults to `$USER` if not provided.

### `colony-relay status`

Check if the relay is running.

```bash
colony-relay status
```

Exit code 0 if running, 1 if not.

## Server discovery

The `say`, `hear`, and `status` commands find the server by:

1. Walking up from the current directory looking for `.colony-relay/port`
2. Falling back to `--server` flag
3. Falling back to `RELAY_SERVER` environment variable

## @mentions

Messages support `@name` mentions. When polling with `--for`, only messages containing that name (or `@all`) are returned. Use `--all` to receive everything.

## Web UI

Open `http://localhost:4100` in a browser for a terminal-style web interface with real-time message streaming.
