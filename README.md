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
curl -L https://github.com/WillyV3/slaygent-comms/archive/v0.1.4.tar.gz | tar xz
cd slaygent-comms-0.1.4
./install.sh
```

## Usage

```bash
slay                          # Launch TUI manager
msg <agent> "message"         # Send message to agent
```

See `CLAUDE.md` for development details.

## Updating

### Option 1: Homebrew Users
```bash
brew upgrade willyv3/tap/slaygent-comms
```

### Option 2: Direct Install Users
Switch to Homebrew for easier updates:
```bash
# Install Homebrew on Linux (if not already installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Then use standard Homebrew update workflow
brew install willyv3/tap/slaygent-comms
brew upgrade willyv3/tap/slaygent-comms
```

## Features

- **TUI Manager** - Visual interface for agent management
- **Message System** - Inter-agent communication via tmux
- **Registry Sync** - Automatically sync agent info to CLAUDE.md files