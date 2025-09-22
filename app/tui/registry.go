package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// RegisteredAgent is a simple registration with name, type, and directory
type RegisteredAgent struct {
	Name      string `json:"name"`      // User-given name
	AgentType string `json:"agent_type"` // claude, opencode, coder, crush
	Directory string `json:"directory"`  // Full working directory path
	Machine   string `json:"machine"`    // Machine name (defaults to "host")
}

// Registry manages the registry.json file
type Registry struct {
	agents   []RegisteredAgent
	filePath string
}

// NewRegistry creates or loads the registry
func NewRegistry() (*Registry, error) {
	// Use ~/.slaygent/registry.json for production
	home, err := os.UserHomeDir()
	registryPath := "registry.json" // fallback to local
	if err == nil {
		slaygentDir := filepath.Join(home, ".slaygent")
		// Create .slaygent directory if it doesn't exist
		os.MkdirAll(slaygentDir, 0755)
		registryPath = filepath.Join(slaygentDir, "registry.json")
	}

	r := &Registry{
		agents:   []RegisteredAgent{},
		filePath: registryPath,
	}

	// Load existing registry if it exists
	r.Load()
	return r, nil
}

// Register adds a new agent with a name
func (r *Registry) Register(name, agentType, directory string) error {
	return r.RegisterWithMachine(name, agentType, directory, "host")
}

// RegisterWithMachine adds a new agent with a name and machine
func (r *Registry) RegisterWithMachine(name, agentType, directory, machine string) error {
	// Remove any existing registration for this type+directory+machine
	r.DeregisterWithMachine(agentType, directory, machine)

	// Add new registration
	r.agents = append(r.agents, RegisteredAgent{
		Name:      name,
		AgentType: agentType,
		Directory: directory,
		Machine:   machine,
	})

	return r.Save()
}

// Deregister removes an agent by type and directory (local machine only)
func (r *Registry) Deregister(agentType, directory string) error {
	return r.DeregisterWithMachine(agentType, directory, "host")
}

// DeregisterWithMachine removes an agent by type, directory, and machine
func (r *Registry) DeregisterWithMachine(agentType, directory, machine string) error {
	filtered := []RegisteredAgent{}
	for _, agent := range r.agents {
		if !(agent.AgentType == agentType && agent.Directory == directory && agent.Machine == machine) {
			filtered = append(filtered, agent)
		}
	}
	r.agents = filtered
	return r.Save()
}

// IsRegistered checks if an agent type+directory has a name (local machine only)
func (r *Registry) IsRegistered(agentType, directory string) bool {
	return r.IsRegisteredWithMachine(agentType, directory, "host")
}

// IsRegisteredWithMachine checks if an agent type+directory+machine has a name
func (r *Registry) IsRegisteredWithMachine(agentType, directory, machine string) bool {
	for _, agent := range r.agents {
		if agent.AgentType == agentType && agent.Directory == directory && agent.Machine == machine {
			return true
		}
	}
	return false
}

// GetName returns the registered name for an agent (local machine only)
func (r *Registry) GetName(agentType, directory string) string {
	return r.GetNameWithMachine(agentType, directory, "host")
}

// GetNameWithMachine returns the registered name for an agent on a specific machine
func (r *Registry) GetNameWithMachine(agentType, directory, machine string) string {
	for _, agent := range r.agents {
		if agent.AgentType == agentType && agent.Directory == directory && agent.Machine == machine {
			return agent.Name
		}
	}
	return ""
}

// GetAgents returns all registered agents
func (r *Registry) GetAgents() []RegisteredAgent {
	return r.agents
}

// Load reads the registry from disk
func (r *Registry) Load() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's ok
		}
		return err
	}

	if err := json.Unmarshal(data, &r.agents); err != nil {
		return err
	}

	// Migrate existing agents without Machine field to "host"
	modified := false
	for i := range r.agents {
		if r.agents[i].Machine == "" {
			r.agents[i].Machine = "host"
			modified = true
		}
	}

	// Save migrated data if needed
	if modified {
		return r.Save()
	}

	return nil
}

// SyncWithActive removes registry entries that don't match any active agents
func (r *Registry) SyncWithActive(activeAgents [][]string) error {
	// Build set of active agent keys (type:directory)
	activeSet := make(map[string]bool)
	for _, row := range activeAgents {
		if len(row) >= 3 {
			agentType := row[2]  // AGENT column
			directory := row[1]  // DIRECTORY column
			key := agentType + ":" + directory
			activeSet[key] = true
		}
	}

	// Filter out agents that are no longer active
	filtered := []RegisteredAgent{}
	for _, agent := range r.agents {
		key := agent.AgentType + ":" + agent.Directory
		if activeSet[key] {
			filtered = append(filtered, agent)
		}
	}

	// Update if anything changed
	if len(filtered) != len(r.agents) {
		r.agents = filtered
		return r.Save()
	}
	return nil
}

// Save writes the registry to disk
func (r *Registry) Save() error {
	data, err := json.MarshalIndent(r.agents, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0644)
}