# Slaygent-Comms Constitution

## Core Principles

### I. Single-File Until Idiomatic (NON-NEGOTIABLE)
Start with everything in main.go and keep it there until splitting becomes natural and necessary. Split by user-facing features, not technical layers. Avoid making many files until the codebase proves it needs them through actual complexity, not anticipated complexity.

### II. No Overengineering (NON-NEGOTIABLE)
Write simple, direct code that solves the current problem. No abstractions until you have 3+ concrete implementations. No interfaces until you have multiple concrete types. No frameworks when a simple function will do. Build what you need now, not what you might need later.

### III. No Fallbacks Ever (NON-NEGOTIABLE)
Write code that works correctly the first time. No try-catch-fallback patterns. No graceful degradation that masks real problems. If tmux isn't running, show clear error. If detection fails, show "unknown" - don't guess. Fail fast and clearly rather than limp along with incorrect behavior.

### IV. Self-Documenting Code (NON-NEGOTIABLE)
Code must explain itself through clear naming, obvious structure, and purposeful organization. Functions do one thing and their name says what. Variables have meaningful names. Control flow is straightforward. Comments only when WHY is not obvious from code.

### V. TUI Responsiveness (NON-NEGOTIABLE)
Every keypress must have immediate visual feedback. Never block the event loop with I/O operations. Use Bubble Tea commands for async work. Render only what changed. Keep Update() functions fast (<1ms). Show loading states for operations >100ms.

## TUI Architecture Standards

### Bubble Tea Patterns
- **Model-Update-View Trinity**: Model holds state only, Update handles transitions, View renders current state
- **Commands for Side Effects**: All I/O operations return commands, never execute directly in Update()
- **Pure Functions**: View() must be deterministic - same state produces same output
- **Single Source of Truth**: All state lives in the model, no globals or hidden state

### Keyboard Navigation
- **Universal Keys**: q/Ctrl+C to quit, Esc to go back, Enter to confirm, ? for help
- **Arrow Navigation**: Up/down for lists, left/right for tabs/pages
- **Mnemonic Keys**: r for refresh, / for search, tab for next field
- **No Mouse Dependency**: Every feature must work with keyboard only

### Performance Requirements
- **Startup Time**: < 500ms from command to first render
- **Refresh Time**: < 100ms for data updates
- **Memory Usage**: < 50MB for normal operation (10-50 tmux sessions)
- **Responsiveness**: < 16ms per frame (60fps event loop)

## Development Workflow

### Test-Driven Development
- **Red-Green-Refactor**: Write failing test → implement minimum code → refactor
- **Contract Tests First**: Test interfaces before implementation
- **Integration Tests**: Test user scenarios end-to-end
- **No Implementation Without Tests**: Every function must have a corresponding test

### Code Organization
- **Single main.go**: Start here and stay here until >300 lines
- **Split by Screen**: When splitting, organize by user-facing features
- **Package main Rule**: Stay in package main until you have reusable components
- **Dependency Direction**: Dependencies flow inward, never circular

### Quality Gates
- **All Tests Pass**: No green CI without 100% test success
- **Go fmt/vet**: Code must pass standard Go tooling
- **Build Success**: Application must compile and run without errors
- **Manual Scenarios**: Core user paths must be manually verified

## Governance

### Constitution Authority
This constitution supersedes all other development practices and preferences. When in doubt, refer to these principles. Complexity must be justified against these rules.

### Amendment Process
Changes to this constitution require:
1. Documentation of why change is needed
2. Approval from project maintainer
3. Migration plan for existing code
4. Update to all related documentation

### Compliance Verification
- All code reviews must verify constitutional compliance
- Automated checks enforce formatting and testing requirements
- Manual verification of user experience principles
- Regular architecture reviews against overengineering

**Version**: 1.0.0 | **Ratified**: 2025-09-16 | **Last Amended**: 2025-09-16