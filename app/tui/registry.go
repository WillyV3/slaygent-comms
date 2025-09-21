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
	// Remove any existing registration for this type+directory
	r.Deregister(agentType, directory)

	// Add new registration
	r.agents = append(r.agents, RegisteredAgent{
		Name:      name,
		AgentType: agentType,
		Directory: directory,
	})

	return r.Save()
}

// Deregister removes an agent by type and directory
func (r *Registry) Deregister(agentType, directory string) error {
	filtered := []RegisteredAgent{}
	for _, agent := range r.agents {
		if !(agent.AgentType == agentType && agent.Directory == directory) {
			filtered = append(filtered, agent)
		}
	}
	r.agents = filtered
	return r.Save()
}

// IsRegistered checks if an agent type+directory has a name
func (r *Registry) IsRegistered(agentType, directory string) bool {
	for _, agent := range r.agents {
		if agent.AgentType == agentType && agent.Directory == directory {
			return true
		}
	}
	return false
}

// GetName returns the registered name for an agent
func (r *Registry) GetName(agentType, directory string) string {
	for _, agent := range r.agents {
		if agent.AgentType == agentType && agent.Directory == directory {
			return agent.Name
		}
	}
	return ""
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

	return json.Unmarshal(data, &r.agents)
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