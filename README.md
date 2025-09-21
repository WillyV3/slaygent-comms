# Slaygent Communication Suite

Inter-agent communication system for AI coding assistants in tmux panes.

## Quick Install

### macOS (Homebrew)
```bash
brew install willyv3/tap/slaygent-comms
```

### Linux (Homebrew)
```bash
brew install willyv3/tap/slaygent-comms
```

### Linux (Direct Install)
```bash
curl -L https://github.com/WillyV3/slaygent-comms/archive/v0.1.1.tar.gz | tar xz
cd slaygent-comms-0.1.1
./install.sh
```

## Usage

```bash
slay                          # Launch TUI manager
msg <agent> "message"         # Send message to agent
```

See `CLAUDE.md` for development details.

## Features

- **TUI Manager** - Visual interface for agent management
- **Message System** - Inter-agent communication via tmux
- **Registry Sync** - Automatically sync agent info to CLAUDE.md files