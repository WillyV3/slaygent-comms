# SSH Stuff

## Cross-Machine Setup

SSH configuration enables agent communication across multiple machines. Agents on remote systems appear in the main table alongside local agents.

## SSH Registry

Remote connections stored in `~/.slaygent/ssh-registry.json`:

```json
[
  {
    "name": "homelab",
    "ssh_key": "~/.ssh/homelab_key",
    "connect_command": "ssh user@192.168.1.100"
  }
]
```

## Adding SSH Connections

1. Press `z` in agents view to add SSH connection
2. Enter machine name (e.g., "homelab")
3. Select SSH key from `~/.ssh/` directory
4. Enter SSH connection command
5. Connection appears in SSH connections view

## SSH Manager

Press `x` in agents view to enter SSH connection manager for viewing and managing existing connections.

## Remote Agent Discovery

System automatically discovers agents on remote machines by querying their `~/.slaygent/registry.json` files. Remote agents appear with machine name in the MACHINE column.

## Cross-Machine Messaging

Use `msg-ssh` for remote messaging:
```bash
msg-ssh <remote_agent> "message content"
msg-ssh --from <sender> <remote_agent> "tracked response"
```

Messages delivered directly to remote tmux panes via SSH without requiring remote msg installation.

## Registry Synchronization

Local registry automatically adopts remote agent registrations during refresh operations. This keeps agent lists consistent across all machines.

## Troubleshooting

- Verify SSH key permissions (600)
- Test SSH connection manually first
- Check remote agent registry: `ssh user@host cat ~/.slaygent/registry.json`
- Ensure tmux running on remote machine