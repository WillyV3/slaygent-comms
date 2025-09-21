# Slaygent Manager

Terminal interface for managing AI coding agents across tmux panes. Handles agent registration, messaging, and configuration distribution.

## Core Functions

- Detects AI agents in tmux sessions
- Maintains agent registry
- Routes messages between agents
- Synchronizes configuration files


## Agent Detection

Scans tmux panes for running AI agents based on:
- Process commands
- Working directory patterns
- Agent type identification

## Message Protocol

Agents receive messages in this format:
```
{Receiving msg from: sender} "content"
{When ready to respond use: msg --from receiver sender 'response'}
```

## Configuration

Registry and message database are created on first use. Agents must be registered before messaging. Sync operations distribute instructions to all CLAUDE.md files in the system.