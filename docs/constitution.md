# Slaygent Communication Suite Constitution

## Core Development Principles

**Simplicity First**
Build minimal, focused solutions. No feature creep. Each component does one thing well.

**Cross-Platform Reliability**
Works consistently on macOS and Linux. Automated dependency management. No manual setup complexity.

**Real-Time Responsiveness**
Live monitoring of tmux sessions. Instant message delivery. Responsive UI that adapts to terminal changes.

**Persistent State**
SQLite for reliable message history. JSON registries for agent discovery. No data loss during system changes.

**Modern Tooling**
Use `fd` over `find`, `rg` over `grep`. Bubble Tea for TUI. Embedded assets for deployment simplicity.

## Architecture Standards

**Multi-Module Design**
Separate Go modules for TUI, messenger, and tests. Each builds independently from correct directory.

**Stateless Views**
UI components receive data and dimensions, return rendered strings. State management separate from rendering.

**Registry-Based Discovery**
Agent identification through working directory matching. Automatic synchronization across machines.

## Communication Protocol

**Structured Messaging**
Consistent message format with sender identification and response instructions. Protocol transparency for AI agents.

**Conversation Tracking**
Bidirectional message logging with timestamps. Conversation grouping for history management.

**SSH Integration**
Remote agent communication without dependency on remote tools. Direct tmux command execution over SSH.

## Quality Standards

**Comprehensive Testing**
Unit tests for core logic. Contract tests for API validation. Integration tests for end-to-end behavior.

**Responsive Design**
Handle terminal resize events. Flexible table columns. Proper viewport sizing for all views.

**Error Resilience**
Graceful fallbacks for missing dependencies. Clear error messages. Recovery from network failures.

## Governance

**User-Centric Development**
Features driven by actual use cases. No theoretical additions. Regular feedback integration.

**Breaking Changes**
Major version bumps for incompatible changes. Migration paths for existing users. Deprecation warnings.

**Documentation Standards**
Concise but comprehensive guides. Embedded help system. Clear troubleshooting instructions.

---
Version: 1.0 | Ratified: 2025-09-21 | Last Amended: 2025-09-21