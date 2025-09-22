package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SSHConnection represents a connection to a remote machine
type SSHConnection struct {
	Name           string `json:"name"`            // User-given name for the machine
	SSHKey         string `json:"ssh_key"`         // Path to SSH key file
	ConnectCommand string `json:"connect_command"` // Full SSH command to connect
}

// SSHRegistry manages the ssh-registry.json file
type SSHRegistry struct {
	machines []SSHConnection
	filePath string
}

// NewSSHRegistry creates or loads the SSH registry
func NewSSHRegistry() (*SSHRegistry, error) {
	// Use ~/.slaygent/ssh-registry.json
	home, err := os.UserHomeDir()
	if err != nil {
		panic("failed to get user home directory for SSH registry")
	}

	slaygentDir := filepath.Join(home, ".slaygent")
	// Create .slaygent directory if it doesn't exist
	os.MkdirAll(slaygentDir, 0755)
	registryPath := filepath.Join(slaygentDir, "ssh-registry.json")

	r := &SSHRegistry{
		machines: []SSHConnection{},
		filePath: registryPath,
	}

	// Load existing registry if it exists
	r.Load()
	return r, nil
}

// Load reads the SSH registry from disk
func (r *SSHRegistry) Load() error {
	data, err := os.ReadFile(r.filePath)
	if os.IsNotExist(err) {
		// File doesn't exist - start with empty registry
		return nil
	}
	if err != nil {
		return err
	}

	var machines []SSHConnection
	if err := json.Unmarshal(data, &machines); err != nil {
		return err
	}

	r.machines = machines
	return nil
}

// Save writes the SSH registry to disk
func (r *SSHRegistry) Save() error {
	data, err := json.MarshalIndent(r.machines, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0644)
}

// AddConnection adds a new SSH connection
func (r *SSHRegistry) AddConnection(name, sshKey, connectCommand string) error {
	// Remove any existing connection with the same name
	r.RemoveConnection(name)

	// Add new connection
	r.machines = append(r.machines, SSHConnection{
		Name:           name,
		SSHKey:         sshKey,
		ConnectCommand: connectCommand,
	})

	return r.Save()
}

// RemoveConnection removes an SSH connection by name
func (r *SSHRegistry) RemoveConnection(name string) error {
	for i, machine := range r.machines {
		if machine.Name == name {
			r.machines = append(r.machines[:i], r.machines[i+1:]...)
			break
		}
	}
	return r.Save()
}

// GetConnection returns an SSH connection by name
func (r *SSHRegistry) GetConnection(name string) *SSHConnection {
	for _, machine := range r.machines {
		if machine.Name == name {
			return &machine
		}
	}
	return nil
}

// GetConnections returns all SSH connections
func (r *SSHRegistry) GetConnections() []SSHConnection {
	return r.machines
}

// ConnectionExists checks if a connection name already exists
func (r *SSHRegistry) ConnectionExists(name string) bool {
	return r.GetConnection(name) != nil
}