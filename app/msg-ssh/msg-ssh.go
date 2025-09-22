package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SSHConnection represents a connection to a remote machine
type SSHConnection struct {
	Name           string `json:"name"`
	SSHKey         string `json:"ssh_key"`
	ConnectCommand string `json:"connect_command"`
}

// RegistryEntry represents a registered agent
type RegistryEntry struct {
	Name      string `json:"name"`
	AgentType string `json:"agent_type"`
	Directory string `json:"directory"`
	Machine   string `json:"machine"`
}

// CrossMachineRegistry combines local agents and SSH connections
type CrossMachineRegistry struct {
	LocalAgents    []RegistryEntry   `json:"local_agents"`
	SSHConnections []SSHConnection   `json:"ssh_connections"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  msg-ssh <agent_name> <message>\n")
		fmt.Fprintf(os.Stderr, "  msg-ssh --from <sender> <agent_name> <message>\n")
		fmt.Fprintf(os.Stderr, "  msg-ssh --status\n")
		fmt.Fprintf(os.Stderr, "  msg-ssh --discover <machine_name>\n")
		os.Exit(1)
	}

	if os.Args[1] == "--status" {
		showCrossMachineStatus()
		os.Exit(0)
	}

	if os.Args[1] == "--discover" && len(os.Args) >= 3 {
		discoverRemoteAgents(os.Args[2])
		os.Exit(0)
	}

	// Parse --from flag if present
	var senderName string
	var agentName string
	var message string

	if len(os.Args) >= 5 && os.Args[1] == "--from" {
		// Format: msg-ssh --from <sender> <receiver> <message>
		senderName = os.Args[2]
		agentName = os.Args[3]
		message = strings.Join(os.Args[4:], " ")
	} else if len(os.Args) >= 3 {
		// Format: msg-ssh <receiver> <message>
		agentName = os.Args[1]
		message = strings.Join(os.Args[2:], " ")
	} else {
		fmt.Fprintf(os.Stderr, "Error: missing message\n")
		os.Exit(1)
	}

	// Load registries
	localRegistry := loadLocalRegistry()
	sshRegistry := loadSSHRegistry()

	if localRegistry == nil || sshRegistry == nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load registries\n")
		os.Exit(1)
	}

	// Find target agent (could be local or remote)
	targetAgent, targetMachine := findAgent(agentName, localRegistry, sshRegistry)
	if targetAgent == nil {
		fmt.Fprintf(os.Stderr, "Error: agent '%s' not found in any registry\n", agentName)
		os.Exit(1)
	}

	// Detect sender if not provided
	if senderName == "" {
		senderName = detectSender(localRegistry)
		if senderName == "" {
			senderName = "unknown"
		}
	}

	// Send message
	if targetMachine == "host" {
		// Local agent - use regular msg tool
		sendLocalMessage(senderName, agentName, message)
	} else {
		// Remote agent - use SSH
		sendRemoteMessage(senderName, agentName, message, targetMachine, sshRegistry)
	}
}

func showCrossMachineStatus() {
	fmt.Println("Cross-Machine Agent Status")
	fmt.Println("==========================")

	// Show local agents
	localRegistry := loadLocalRegistry()
	if localRegistry != nil {
		fmt.Printf("\nLocal Agents (host):\n")
		for _, agent := range localRegistry {
			fmt.Printf("  %s (%s) - %s\n", agent.Name, agent.AgentType, agent.Directory)
		}
	}

	// Show SSH connections
	sshRegistry := loadSSHRegistry()
	if sshRegistry != nil {
		fmt.Printf("\nSSH Connections:\n")
		for _, conn := range sshRegistry {
			fmt.Printf("  %s - %s\n", conn.Name, conn.ConnectCommand)
		}
	}

	// Show remote agents
	if sshRegistry != nil {
		for _, conn := range sshRegistry {
			fmt.Printf("\nAgents on %s:\n", conn.Name)
			remoteAgents := queryRemoteAgents(conn)
			if len(remoteAgents) == 0 {
				fmt.Printf("  (none found or connection failed)\n")
			} else {
				for _, agent := range remoteAgents {
					fmt.Printf("  %s (%s) - %s\n", agent.Name, agent.AgentType, agent.Directory)
				}
			}
		}
	}
}

func discoverRemoteAgents(machineName string) {
	sshRegistry := loadSSHRegistry()
	if sshRegistry == nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load SSH registry\n")
		os.Exit(1)
	}

	var targetConn *SSHConnection
	for _, conn := range sshRegistry {
		if conn.Name == machineName {
			targetConn = &conn
			break
		}
	}

	if targetConn == nil {
		fmt.Fprintf(os.Stderr, "Error: SSH connection '%s' not found\n", machineName)
		os.Exit(1)
	}

	fmt.Printf("Discovering agents on %s...\n", machineName)
	agents := queryRemoteAgents(*targetConn)

	if len(agents) == 0 {
		fmt.Printf("No agents found on %s\n", machineName)
	} else {
		fmt.Printf("Found %d agents on %s:\n", len(agents), machineName)
		for _, agent := range agents {
			fmt.Printf("  %s (%s) - %s\n", agent.Name, agent.AgentType, agent.Directory)
		}
	}
}

func findAgent(name string, localRegistry []RegistryEntry, sshRegistry []SSHConnection) (*RegistryEntry, string) {
	// Check local registry first
	for _, agent := range localRegistry {
		if agent.Name == name {
			return &agent, "host"
		}
	}

	// Check remote registries
	for _, conn := range sshRegistry {
		remoteAgents := queryRemoteAgents(conn)
		for _, agent := range remoteAgents {
			if agent.Name == name {
				return &agent, conn.Name
			}
		}
	}

	return nil, ""
}

func sendLocalMessage(sender, receiver, message string) {
	// Use the regular msg tool for local messaging
	var cmd *exec.Cmd
	if sender != "unknown" {
		cmd = exec.Command("msg", "--from", sender, receiver, message)
	} else {
		cmd = exec.Command("msg", receiver, message)
	}

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending local message: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Message sent to %s (local)\n", receiver)
}

func sendRemoteMessage(sender, receiver, message, machine string, sshRegistry []SSHConnection) {
	var targetConn *SSHConnection
	for _, conn := range sshRegistry {
		if conn.Name == machine {
			targetConn = &conn
			break
		}
	}

	if targetConn == nil {
		fmt.Fprintf(os.Stderr, "Error: SSH connection for machine '%s' not found\n", machine)
		os.Exit(1)
	}

	// Build SSH command
	sshParts := strings.Fields(targetConn.ConnectCommand)
	if len(sshParts) == 0 {
		fmt.Fprintf(os.Stderr, "Error: invalid SSH connect command: %s\n", targetConn.ConnectCommand)
		os.Exit(1)
	}

	// Add SSH key if specified
	if targetConn.SSHKey != "" {
		expandedKey := expandPath(targetConn.SSHKey)
		sshParts = append(sshParts[:1], append([]string{"-i", expandedKey}, sshParts[1:]...)...)
	}

	// Build remote msg command
	var remoteMsgCmd string
	if sender != "unknown" {
		remoteMsgCmd = fmt.Sprintf("msg --from %s %s '%s'", sender, receiver, message)
	} else {
		remoteMsgCmd = fmt.Sprintf("msg %s '%s'", receiver, message)
	}

	// Execute SSH command
	fullCmd := append(sshParts, remoteMsgCmd)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, fullCmd[0], fullCmd[1:]...)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending remote message to %s: %v\n", machine, err)
		os.Exit(1)
	}

	fmt.Printf("Message sent to %s on %s\n", receiver, machine)
}

func queryRemoteAgents(conn SSHConnection) []RegistryEntry {
	// Build SSH command to query remote registry
	sshParts := strings.Fields(conn.ConnectCommand)
	if len(sshParts) == 0 {
		return nil
	}

	// Add SSH key if specified
	if conn.SSHKey != "" {
		expandedKey := expandPath(conn.SSHKey)
		sshParts = append(sshParts[:1], append([]string{"-i", expandedKey}, sshParts[1:]...)...)
	}

	// Query remote registry
	remoteCmd := "cat ~/.slaygent/registry.json 2>/dev/null || echo '[]'"
	fullCmd := append(sshParts, remoteCmd)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, fullCmd[0], fullCmd[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var agents []RegistryEntry
	if err := json.Unmarshal(output, &agents); err != nil {
		return nil
	}

	return agents
}

func loadLocalRegistry() []RegistryEntry {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	registryPath := filepath.Join(home, ".slaygent", "registry.json")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil
	}

	var registry []RegistryEntry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil
	}

	return registry
}

func loadSSHRegistry() []SSHConnection {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	registryPath := filepath.Join(home, ".slaygent", "ssh-registry.json")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil // File might not exist yet
	}

	var registry []SSHConnection
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil
	}

	return registry
}

func detectSender(localRegistry []RegistryEntry) string {
	// Try to detect sender from current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	for _, agent := range localRegistry {
		if agent.Directory == cwd {
			return agent.Name
		}
	}

	return ""
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}