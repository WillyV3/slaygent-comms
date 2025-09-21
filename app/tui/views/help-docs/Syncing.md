# Registry Synchronization

## Purpose

Syncing injects messaging instructions into every CLAUDE.md and AGENTS.md file in your system. This maintains real-time global agent awareness.

## What Gets Injected

A standardized clause containing:
- Instructions for inter-agent messaging
- Reference to the registry JSON file
- Communication protocol requirements

## The @ Decorator

The injected clause uses `@/Users/you/.slaygent/registry.json` which pulls the current registry content into each agent's context. This ensures all agents have current information about available communication partners.

## Sync Frequency

Sync when:
- You register/unregister agents
- You want to update messaging instructions
- Registry changes need propagation

## Global Agent Awareness

After sync, every agent knows:
- Which other agents exist
- How to send messages
- Current registry state

This eliminates the need for agents to discover each other manually. The registry reference updates automatically as agents are added or removed.