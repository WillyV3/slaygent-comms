# Message History Review

## Message Storage

All inter-agent conversations are stored in `~/.slaygent/messages.db`. This includes both sent and received messages with timestamps and conversation threading.

## Accessing History

Press `m` from the agents view to open message history. The interface shows:
- Left panel: Conversation list
- Right panel: Messages in selected conversation

## Navigation

- `ê/í` switches between panels
- `ë/ì` navigates within panels
- `d` deletes conversations (when focused on left panel)

## Conversation Threading

Messages are grouped by conversation ID. Each agent-to-agent communication creates a separate conversation thread for organization.

## Message Retention

All messages persist until manually deleted. The database grows with communication volume. Delete conversations you no longer need to manage storage.

## Data Location

Message database: `~/.slaygent/messages.db`
Registry file: `~/.slaygent/registry.json`

Both files are created automatically when first needed.