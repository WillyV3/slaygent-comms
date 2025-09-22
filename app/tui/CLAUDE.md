# Slaygent TUI Manager

Responsive Bubble Tea UI for AI agent management across local and SSH machines.

## Quick Start
```bash
slay                    # Launch TUI (installed via Homebrew)
go build -o slaygent-manager  # Build from app/tui/ directory
```

## Core Architecture
```
Bubble Tea TUI ↔ Views (Stateless) ↔ Data (Registry/DB/Tmux)
```
```

## Navigation
- **Agents View**: `r` refresh, `m` messages, `s` sync, `?` help, `a` register
- **Messages View**: `←/→` panels, `d` delete, `ESC` back
- **Help View**: `←/→` tabs, `↑/↓` scroll, `ESC` back

## Key Features
- **Cross-machine discovery** - Local + SSH agent monitoring
- **Auto-registry adoption** - Syncs remote agent registrations
- **Responsive layout** - Terminal resize handling
- **Bubble table** - Modern text handling with flex columns
- **Embedded help** - Tabbed markdown documentation

## Components
- **main.go** - State management, responsive resize handling
- **views/** - Stateless rendering (agents, messages, sync, help)
- **registry.go** - Agent registration and tmux synchronization
- **tmux.go** - Local/remote agent discovery
- **history/** - SQLite message persistence

## Development Notes
- **Responsive Design** - Handle `tea.WindowSizeMsg` for proper layout
- **Bubble Table** - Use `NewStyledCell()` for individual cell styling
- **Embedded Help** - Go embed for shipping help docs with binary
- **Modern Tools** - Always use `fd` over `find`, `rg` over `grep`

## Data Integration
- **Registry Sync** - Auto-removes stale entries, adopts remote agents
- **Message History** - SQLite integration with conversation grouping
- **SSH Support** - Remote machine agent discovery and management

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
