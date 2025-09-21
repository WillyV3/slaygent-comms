# Agent Registration

## What Registration Does

Registering an agent adds it to the global registry file at `~/.slaygent/registry.json`. This makes the agent discoverable for inter-agent communication.

## Registry Structure

The JSON registry contains:
- Agent name (your chosen identifier)
- Agent type (claude, opencode, coder, etc.)
- Working directory path

```json
[
  {
    "name": "backend-dev",
    "agent_type": "claude",
    "directory": "/Users/you/project-backend"
  }
]
```

## Registration Process

1. Navigate to an agent in the table
2. Press `a` to register/unregister
3. Enter a name when prompted
4. The agent gets added to the registry

Unregistered agents cannot send or receive messages. The registry is synchronized across all CLAUDE.md files during sync operations.

## Registry Updates

Changes to the registry are immediately available to the messaging system. All agents reference the same registry file through the `@` decorator in their CLAUDE.md files.