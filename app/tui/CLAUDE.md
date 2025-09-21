# Slaygent TUI Manager

Bubble Tea terminal UI for agent management and messaging with tabbed help system.

## Build & Run
```bash
# Build from /tui directory
go build -o slaygent-manager
./slaygent-manager

# Window title automatically set to "Slaygent Manager"
```

## Architecture Map
```
┌──────────────────────────────────────┐
│         Bubble Tea TUI               │
│  ┌─────────────────────────────┐     │
│  │     main.go (Model)         │     │
│  │  - State management         │     │
│  │  - Update() logic           │     │
│  │  - Window resize handling   │     │
│  └────────────┬────────────────┘     │
│               ▼                       │
│  ┌─────────────────────────────┐     │
│  │    views/ (Stateless)       │     │
│  │  - agents.go                │     │
│  │  - messages.go              │     │
│  │  - sync.go                  │     │
│  │  - help.go (tabbed)         │     │
│  │  - help-docs/*.md           │     │
│  └─────────────────────────────┘     │
└────────────────┬─────────────────────┘
                 ▼
    ┌────────────────────────────┐
    │  External Components       │
    │  - registry.json           │
    │  - messages.db             │
    │  - tmux sessions           │
    └────────────────────────────┘
```

## Control Flow
```
Agents View ──┬── e → Sync Editor View
              ├── s → Quick Sync (progress)
              ├── m → Messages View
              ├── ? → Help View (tabbed)
              └── r → Refresh Registry

Messages View ─┬── d → Delete Conversation
               └── ESC → Agents View

Sync View ─────┬── Tab → Edit Mode
               ├── c → Custom Sync → Agents
               └── ESC → Agents View

Help View ─────┬── ←/→ → Switch Tabs
               ├── ↑/↓ → Scroll Content
               └── ESC → Agents View
```

## Model State
```go
type model struct {
    table            table.Model  // Now uses bubble-table
    registry         *Registry
    viewMode         string  // "agents", "messages", "sync", "help"
    historyModel     *history.Model
    syncEditor       textarea.Model
    helpModel        *views.HelpModel
    syncing          bool
    progress         progress.Model
    width, height    int  // Responsive dimensions
}
```

## View Pattern
- main.go handles state/logic and responsive resize events
- views/* render stateless UI components
- Registry syncs with tmux sessions
- Progress animation during sync operations
- Help system uses embedded markdown files for scalability

## Help System
- **Tabbed Interface**: Overview, Registering, Syncing, Inter-Agent Messaging, Messages, About
- **Responsive**: Adapts to terminal height/width changes via `tea.WindowSizeMsg`
- **Embedded Content**: Uses Go embed for `help-docs/*.md` files
- **Navigation**: `←/→` for tabs, `↑/↓` for scrolling, `?` to open from agents view

## File Structure
```
views/
├── help.go              # Tabbed help implementation
├── help-docs/
│   ├── overview.md      # System overview
│   ├── registering.md   # Agent registration
│   ├── Syncing.md       # Registry synchronization
│   ├── messaging.md     # Inter-agent messaging
│   ├── stored-convos.md # Message history
│   └── about.md         # Author and purpose
├── agents.go            # Agent table view (bubble-table)
├── messages.go          # Message history view
└── sync.go              # Sync customization view
```

## Styling
- Baby blue (#87) - Headers/borders/highlights
- ANSI 256 colors - Agent names (hash-based assignment)
- Green - Registered agents
- Red (#FF6B6B) - Warnings/errors
- Simple tab styling - Active (bold, highlighted), Inactive (muted)

## Development Practices
- **Modern CLI Tools**: Use `fd` instead of `find`, `rg` instead of `grep`
- **Responsive Design**: Always handle `tea.WindowSizeMsg` for proper layout
- **Embedded Assets**: Use Go embed for static content that needs to ship with binary
- **Message Flow**: Follow Bubble Tea patterns - fast Update(), stateless View()

## Recent Changes
- **Bubble Table Migration**: Switched from lipgloss table to bubble-table for better text handling
- **Flex Columns**: DIRECTORY and NAME columns now resize with terminal width
- **Messages View Fix**: Viewport dimensions properly initialized on view switch
- **Help Documentation**: Restructured with focused content on core functionality
- **Agent Colors**: Fixed styling in bubble-table using NewStyledCell()

## Development Pitfalls
- **Table Selection**: bubble-table manages selection internally via GetHighlightedRowIndex()
- **Viewport Sizing**: Must set dimensions before entering messages view to prevent layout issues
- **Cell Styling**: Use NewStyledCell() for individual cells, not row-level styling in bubble-table
- **Help Embed**: File paths in help.go must match actual markdown files in help-docs/

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
