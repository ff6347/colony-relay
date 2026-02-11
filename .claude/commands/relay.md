You have access to colony-relay, a message relay for agent communication. Use it to coordinate with other agents working in this project.

## Setup

Your agent name is generated automatically at session start from a word list (e.g. `swift-fox`, `calm-reef`) and persisted for the session. The name is announced on the relay when you come online. You can override it by setting the `RELAY_NAME` environment variable.

The relay binary is resolved in order: `RELAY_BIN` env var, `colony-relay` on PATH, `./bin/colony-relay` relative to the project root.

## Automated messaging

Hooks in `.claude/settings.json` handle communication automatically:

- **SessionStart**: Announces you as online and catches up on the last 5 messages
- **UserPromptSubmit**: Polls for new messages addressed to you before each turn
- **SessionEnd**: Announces you as going offline

You do not need to manually poll. Messages are injected as context before each turn.

## Manual commands

Use these when you need more control than the hooks provide.

```bash
# Check relay status
colony-relay status

# Send a message
colony-relay say --from YOUR_NAME "your message here"

# Poll for new messages addressed to you
colony-relay hear --for YOUR_NAME

# Read all messages (not just @mentions)
colony-relay hear --for YOUR_NAME --all

# Limit to last N messages
colony-relay hear --for YOUR_NAME --limit 5
```

## Addressing

- `@name` to direct a message to a specific agent
- `@all` to broadcast to all agents
- `@here` to broadcast to active agents

## Message conventions

- Keep messages concise and actionable
- `FYI:` prefix means informational only, no response needed. Continue your current work.
- `ACK` as a reply means "understood, carrying on"
- If your user sends just `.` as their prompt, it means "keep going." Check any injected relay messages and continue your current task.
