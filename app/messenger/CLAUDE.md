# Slaygent Messenger

CLI tool for inter-agent messaging via tmux panes.

## Build & Usage
```bash
go build -o msg
msg <agent_name> "message"
msg --from <sender> <receiver> "response"
msg --status
```

## Message Protocol
Incoming format:
```
{Receiving msg from: <sender>} "<message>"
{When ready to respond use: msg --from <receiver> <sender> 'response'}
```

## Architecture Map
```
┌─────────────────┐     ┌──────────────────┐
│   msg CLI       │────▶│  Registry JSON   │
│  (msg.go)       │     │ (~/.slaygent/)   │
└────────┬────────┘     └──────────────────┘
         │
         ▼
┌─────────────────┐     ┌──────────────────┐
│  SQLite DB      │────▶│  Agent Terminal  │
│ (messages.db)   │     │  (tmux pane)     │
└─────────────────┘     └──────────────────┘
```

## Components
- **msg.go** - CLI interface, registry-based sender detection
- **Database** - SQLite at `~/.slaygent/messages.db`
- **Registry** - JSON at `~/.slaygent/registry.json`

## Message Flow & Data Pipeline
```
1. SEND:   msg <agent> "text"
           ↓
2. DETECT: Registry lookup (sender from cwd)
           ↓
3. STORE:  SQLite (conversation + message)
           ↓
4. ROUTE:  Find agent tmux pane
           ↓
5. DELIVER: Send to pane with protocol format
```

## Database Schema
```sql
conversations: id, agent1_name, agent2_name, last_message_at
messages: id, conversation_id, sender_name, receiver_name, message, sent_at
```

## Development Tips - Database Inspection
```bash
# Open the message database
sqlite3 ~/.slaygent/messages.db

# Useful SQLite commands:
.tables                    # List all tables
.schema messages          # Show table structure
.headers on               # Show column names
.mode column              # Pretty output

# View recent messages
SELECT * FROM messages ORDER BY sent_at DESC LIMIT 10;

# Check conversations
SELECT * FROM conversations;

# Debug specific agent messages
SELECT sender_name, receiver_name, message, datetime(sent_at, 'localtime')
FROM messages WHERE sender_name='test-agent' OR receiver_name='test-agent';

# Exit SQLite
.quit
```

## Key Features
- **Registry-based sender detection** - No tmux dependency for basic messaging
- **Directory-independent messaging** - Works after first contact established
- **Full conversation history tracking** - Persistent SQLite storage
- **Automatic agent discovery** - Integration with TUI manager
- **Protocol transparency** - Clear message format for AI agents

## Integration with TUI Manager
- Messages viewable in tabbed help system (`slay` → `?` → Messaging tab)
- Real-time conversation management via TUI (`slay` → `m`)
- Registry synchronization for agent discovery
- Delete conversations through TUI interface

## Color System
- ANSI 256 colors for agent names
- Hash-based consistent assignment
- Vibrant distinct colors per agent

## Troubleshooting
Use the TUI help system (`slay` → `?` → Messaging tab) for comprehensive troubleshooting guides, or check the following:

- **Agent not found**: Verify with `msg --status` or check TUI registry
- **Messages not delivering**: Ensure tmux pane accessibility
- **Database issues**: Check permissions on `~/.slaygent/messages.db`

<!-- SLAYGENT-REGISTRY-START -->
# Inter-Agent Communication
@/home/wv3/.slaygent/registry.json

To send messages to other coding agents, use: `msg <agent_name> "<message>"`
Example: `msg backend-dev "Please update the API endpoint"`

IMPORTANT: When responding to messages, always use the --from flag:
`msg --from <your_agent_name> <target_agent> "<response>"`
This ensures proper conversation logging and tracking.

<!-- Registry automatically synced by slaygent-manager -->
<!-- SLAYGENT-REGISTRY-END -->
