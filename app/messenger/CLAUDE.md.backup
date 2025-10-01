# Slaygent Messenger

CLI messaging tool for AI agents via tmux panes with SQLite persistence.

## Quick Commands
```bash
msg <agent> "message"              # Send message
msg --from <sender> <target> "reply"  # Tracked response
msg --status                       # Show all agents
```

## Message Flow
```
CLI → Registry Lookup → SQLite Log → Tmux Delivery → Agent Terminal
```

## Core Features
- **Registry-based routing** - Automatic sender detection from working directory
- **SQLite persistence** - Full conversation history with timestamps
- **Protocol formatting** - Structured messages with response instructions
- **Cross-machine support** - Works with `msg-ssh` for remote agents

## Data Storage
- **Registry**: `~/.slaygent/registry.json` - Agent discovery
- **Messages**: `~/.slaygent/messages.db` - Conversation history
- **Schema**: conversations ↔ messages (1:many relationship)

## Message Protocol
Received format:
```
{Receiving msg from: <sender>} "<message>"
{When ready to respond use: msg --from <receiver> <sender> 'response'}
```

## Database Debug
```bash
sqlite3 ~/.slaygent/messages.db
.headers on; .mode column
SELECT * FROM messages ORDER BY sent_at DESC LIMIT 5;
```

## Integration
- **TUI Manager**: Message history via `slay` → `m`
- **SSH Support**: Remote agents via `msg-ssh`
- **Registry Sync**: Auto-discovery in distributed environments

<!-- SLAYGENT-REGISTRY-START -->
# Inter-Agent Communication
@~/.slaygent/registry.json

To send messages to other coding agents, use: `msg <agent_name> "<message>"`
Example: `msg backend-dev "Please update the API endpoint"`

IMPORTANT: When responding to messages, always use the --from flag:
`msg --from <your_agent_name> <target_agent> "<response>"`
This ensures proper conversation logging and tracking.

<!-- Registry automatically synced by slaygent-manager -->
<!-- SLAYGENT-REGISTRY-END -->