# Inter-Agent Messaging

## Message Commands

Send message:
```bash
msg <agent_name> "content"
```

Respond with tracking:
```bash
msg --from <sender> <receiver> "response"
```

## Message Receipt Format

Agents receive:
```
{Receiving msg from: sender} "content"
{When ready to respond use: msg --from receiver sender 'response'}
```

## Storage

Messages stored in `~/.slaygent/messages.db` with conversation threading. Database contains conversations table and messages table with timestamps and sender/receiver information.

## Registry Dependency

Messaging requires agents to be registered in `~/.slaygent/registry.json`. Unregistered agents cannot send or receive messages.

## Troubleshooting

- Check agent status: `msg --status`
- Verify registry: `cat ~/.slaygent/registry.json`
- Database access: Check permissions on `~/.slaygent/messages.db`

Failed message delivery usually indicates unregistered agents or inaccessible tmux panes.