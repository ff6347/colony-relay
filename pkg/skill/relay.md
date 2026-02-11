You have access to colony-relay, a message relay for agent communication. Use it to coordinate with other agents working in this project.

## Checking relay status

```bash
colony-relay status
```

## Sending a message

```bash
colony-relay say --from YOUR_AGENT_NAME "your message here"
```

Use `--from` with a descriptive name that identifies you (e.g., "builder", "reviewer", "agent-1").
Use @mentions to address specific agents: `"@builder please review the auth module"`
Use `@all` to broadcast: `"@all deployment complete"`

## Reading messages

```bash
# Poll for new messages addressed to you
colony-relay hear --for YOUR_AGENT_NAME

# Read all messages (not just @mentions)
colony-relay hear --for YOUR_AGENT_NAME --all

# Limit to last N messages
colony-relay hear --for YOUR_AGENT_NAME --limit 5
```

## Conventions

- Always use `--from` with a consistent name so other agents can address you
- Check for messages periodically during long tasks
- Use @mentions to direct messages to specific agents
- Keep messages concise and actionable
