eady to code?                                                                                          │
│                                                                                                         │
│ Here is Claude's plan:                                                                                  │
│ ╭─────────────────────────────────────────────────────────────────────────────────────────────────────╮ │
│ │ SSH Cross-Machine Agent Communication Feature Plan                                                  │ │
│ │                                                                                                     │ │
│ │ Overview                                                                                            │ │
│ │                                                                                                     │ │
│ │ Extend the Slaygent system to support SSH-based inter-machine agent communication. This adds remote │ │
│ │  agent registration and messaging across computers while maintaining the existing local             │ │
│ │ architecture.                                                                                       │ │
│ │                                                                                                     │ │
│ │ Architecture Design                                                                                 │ │
│ │                                                                                                     │ │
│ │ 1. SSH Registry System                                                                              │ │
│ │                                                                                                     │ │
│ │ - New file: ~/.slaygent/ssh-registry.json                                                           │ │
│ │ - Structure:                                                                                        │ │
│ │ {                                                                                                   │ │
│ │   "machines": [                                                                                     │ │
│ │     {                                                                                               │ │
│ │       "name": "linux-dev",                                                                          │ │
│ │       "ssh_key": "~/.ssh/id_rsa",                                                                   │ │
│ │       "connect_command": "ssh user@192.168.1.100"                                                   │ │
│ │     }                                                                                               │ │
│ │   ]                                                                                                 │ │
│ │ }                                                                                                   │ │
│ │                                                                                                     │ │
│ │ 2. Enhanced Agent Registry                                                                          │ │
│ │                                                                                                     │ │
│ │ - Extend existing: ~/.slaygent/registry.json                                                        │ │
│ │ - Add machine field:                                                                                │ │
│ │ {                                                                                                   │ │
│ │   "name": "remote-claude",                                                                          │ │
│ │   "agent_type": "claude",                                                                           │ │
│ │   "directory": "/home/user/project",                                                                │ │
│ │   "machine": "linux-dev"  // NEW field (defaults to "host")                                         │ │
│ │ }                                                                                                   │ │
│ │                                                                                                     │ │
│ │ 3. Cross-Machine Messaging Tool                                                                     │ │
│ │                                                                                                     │ │
│ │ - New file: app/messenger/msg-ssh.go                                                                │ │
│ │ - Functionality: Combines SSH registry + agent registry for remote messaging                        │ │
│ │ - SSH execution: Uses ssh -i <key> <machine> "msg <agent> '<message>'"                              │ │
│ │                                                                                                     │ │
│ │ 4. TUI Enhancements                                                                                 │ │
│ │                                                                                                     │ │
│ │ - agents.go: Add "Machine" column to table                                                          │ │
│ │ - New view: SSH connection registration interface                                                   │ │
│ │ - Agent discovery: Detect remote tmux sessions via SSH                                              │ │
│ │ - Registration flow: Register remote agents from TUI                                                │ │
│ │                                                                                                     │ │
│ │ Implementation Steps                                                                                │ │
│ │                                                                                                     │ │
│ │ Phase 1: SSH Registry Infrastructure                                                                │ │
│ │                                                                                                     │ │
│ │ 1. Create SSH registry JSON structure and operations                                                │ │
│ │ 2. Add SSH connection management functions                                                          │ │
│ │ 3. Build SSH registry loading/saving                                                                │ │
│ │                                                                                                     │ │
│ │ Phase 2: Cross-Machine Messaging                                                                    │ │
│ │                                                                                                     │ │
│ │ 1. Create msg-ssh.go CLI tool                                                                       │ │
│ │ 2. Implement SSH command execution with key authentication                                          │ │
│ │ 3. Add remote agent discovery via SSH tmux queries                                                  │ │
│ │ 4. Integrate with existing message database                                                         │ │
│ │                                                                                                     │ │
│ │ Phase 3: TUI Integration                                                                            │ │
│ │                                                                                                     │ │
│ │ 1. Add "Machine" column to agents table view                                                        │ │
│ │ 2. Create SSH connection registration interface                                                     │ │
│ │ 3. Update agent discovery to include remote machines                                                │ │
│ │ 4. Add SSH agent registration workflow                                                              │ │
│ │                                                                                                     │ │
│ │ Phase 4: Agent Discovery Enhancement                                                                │ │
│ │                                                                                                     │ │
│ │ 1. Extend tmux detection to query remote machines                                                   │ │
│ │ 2. Show remote agents in main agents table                                                          │ │
│ │ 3. Support registration/unregistration of remote agents                                             │ │
│ │ 4. Handle SSH connection failures gracefully                                                        │ │
│ │                                                                                                     │ │
│ │ Key Design Principles                                                                               │ │
│ │                                                                                                     │ │
│ │ Following "No Fallbacks" Rule                                                                       │ │
│ │                                                                                                     │ │
│ │ - SSH failures must be obvious and unignorable                                                      │ │
│ │ - No silent defaults for missing SSH connections                                                    │ │
│ │ - Clear error messages for connection/authentication issues                                         │ │
│ │                                                                                                     │ │
│ │ Maintaining Existing Architecture                                                                   │ │
│ │                                                                                                     │ │
│ │ - Local registry (registry.json) remains unchanged structurally                                     │ │
│ │ - Existing msg tool continues to work for local agents                                              │ │
│ │ - TUI views follow established stateless pattern                                                    │ │
│ │ - Database schema unchanged (machine info in registry only)                                         │ │
│ │                                                                                                     │ │
│ │ Security Considerations                                                                             │ │
│ │                                                                                                     │ │
│ │ - SSH key paths stored as user-provided strings                                                     │ │
│ │ - No automatic key generation or management                                                         │ │
│ │ - User responsible for SSH key access and permissions                                               │ │
│ │ - Clear distinction between local and remote agents in UI                                           │ │
│ │                                                                                                     │ │
│ │ File Changes Required                                                                               │ │
│ │                                                                                                     │ │
│ │ New Files                                                                                           │ │
│ │                                                                                                     │ │
│ │ - ~/.slaygent/ssh-registry.json (runtime)                                                           │ │
│ │ - app/messenger/msg-ssh.go (new CLI tool)                                                           │ │
│ │                                                                                                     │ │
│ │ Modified Files                                                                                      │ │
│ │                                                                                                     │ │
│ │ - app/tui/registry.go (add machine field support)                                                   │ │
│ │ - app/tui/views/agents.go (add Machine column)                                                      │ │
│ │ - app/tui/main.go (add SSH registration view mode)                                                  │ │
│ │ - app/tui/tmux.go (extend for remote detection)                                                     │ │
│ │                                                                                                     │ │
│ │ No Breaking Changes                                                                                 │ │
│ │                                                                                                     │ │
│ │ - Existing msg tool unchanged                                                                       │ │
│ │ - Existing agent registry compatible (machine defaults to "host")                                   │ │
│ │ - Current TUI workflows preserved                                                                   │ │
│ │ - Database schema unchanged                                                                         │ │
│ │                                                                                                     │ │
│ │ This design maintains the system's simplicity while adding powerful cross-machine capabilities      │ │
│ │ following established Bubble Tea patterns and the project's "fail fast" philosophy.                 │ │
│ ╰─────────────────────────────────────────────────────────────────────────────────────────────────────╯ │
│                                                                                                         │
