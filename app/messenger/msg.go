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

type RegistryEntry struct {
	Name      string `json:"name"`
	AgentType string `json:"agent_type"`
	Directory string `json:"directory"`
}

type Pane struct {
	ID        string // session:window.pane
	Command   string
	Directory string
}

func main() {
	// Initialize database
	if err := InitDB(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: database initialization failed: %v\n", err)
		// Continue without logging
	}
	defer CloseDB()

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage:\n  msg <agent_name> <message>\n  msg --from <sender> <agent_name> <message>\n  msg --status\n")
		os.Exit(1)
	}

	if os.Args[1] == "--status" {
		showStatus()
		os.Exit(0)
	}

	// Parse --from flag if present
	var senderName string
	var agentName string
	var message string

	if len(os.Args) >= 5 && os.Args[1] == "--from" {
		// Format: msg --from <sender> <receiver> <message>
		senderName = os.Args[2]
		agentName = os.Args[3]
		message = strings.Join(os.Args[4:], " ")
	} else if len(os.Args) >= 3 {
		// Format: msg <receiver> <message>
		agentName = os.Args[1]
		message = strings.Join(os.Args[2:], " ")
	} else {
		fmt.Fprintf(os.Stderr, "Error: missing message\n")
		fmt.Fprintf(os.Stderr, "Usage: msg <agent_name> <message>\n")
		os.Exit(1)
	}

	// Load registry
	registry := loadRegistry()
	if registry == nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load registry\n")
		os.Exit(1)
	}

	// Find agent
	var targetAgent *RegistryEntry
	for _, agent := range registry {
		if agent.Name == agentName {
			targetAgent = &agent
			break
		}
	}

	if targetAgent == nil {
		fmt.Fprintf(os.Stderr, "Error: agent '%s' not found in registry\n", agentName)
		fmt.Fprintln(os.Stderr, "Registered agents:")
		for _, agent := range registry {
			fmt.Fprintf(os.Stderr, "  - %s\n", agent.Name)
		}
		os.Exit(1)
	}

	// Find pane - ALWAYS use directory-based search for correctness
	// Previous optimization using findAgentPaneByType() for established conversations
	// caused misrouting when multiple agents of the same type were active
	var pane *Pane
	pane = findAgentPane(targetAgent)
	if pane == nil {
		fmt.Fprintf(os.Stderr, "Error: %s (%s) not found in %s\n",
			targetAgent.Name, targetAgent.AgentType, targetAgent.Directory)
		os.Exit(1)
	}

	// Send message
	if sendMessage(pane.ID, message, targetAgent, registry) {
		fmt.Printf("Message sent to %s\n", agentName)

		// Log message to database
		if senderName != "" {
			// Use explicitly provided sender name
			if err := LogMessageExplicit(senderName, targetAgent, message, registry); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log message: %v\n", err)
			}
		} else {
			// Detect sender from current working directory and registry
			senderInfo := detectSenderFromRegistry(registry)
			if senderInfo != "" && senderInfo != "unknown" {
				if err := LogMessageFromRegistry(senderInfo, targetAgent, message, registry); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log message: %v\n", err)
				}
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "Failed to send message to %s\n", agentName)
		os.Exit(1)
	}
}

func loadRegistry() []RegistryEntry {
	// Use ~/.slaygent/registry.json for production
	home, _ := os.UserHomeDir()
	registryPath := filepath.Join(home, ".slaygent", "registry.json")

	data, err := os.ReadFile(registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading registry from %s: %v\n", registryPath, err)
		return nil
	}

	var registry []RegistryEntry
	if err := json.Unmarshal(data, &registry); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing registry: %v\n", err)
		return nil
	}

	return registry
}

func getTmuxPanes() []Pane {
	cmd := exec.Command("tmux", "list-panes", "-a", "-F",
		"#{session_name}:#{window_index}.#{pane_index}:#{pane_current_command}:#{pane_current_path}")

	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var panes []Pane
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 4 {
			continue
		}

		panes = append(panes, Pane{
			ID:        parts[0] + ":" + parts[1],
			Command:   parts[2],
			Directory: parts[3],
		})
	}

	return panes
}

func findAgentPane(agent *RegistryEntry) *Pane {
	panes := getTmuxPanes()

	// First try exact directory match (preferred)
	for _, pane := range panes {
		if pane.Directory == agent.Directory {
			// Check command match
			detectedType := detectAgentType(pane.Command)
			if detectedType == agent.AgentType {
				return &pane
			}

			// For node processes, check deeper
			if pane.Command == "node" {
				actualType := detectNodeAgent(pane.ID)
				if actualType == agent.AgentType {
					return &pane
				}
			}
		}
	}

	// If not found in exact directory, search in any subdirectory
	for _, pane := range panes {
		// Check if pane is in a subdirectory of the registered directory
		if strings.HasPrefix(pane.Directory, agent.Directory) {
			detectedType := detectAgentType(pane.Command)
			if detectedType == agent.AgentType {
				return &pane
			}

			if pane.Command == "node" {
				actualType := detectNodeAgent(pane.ID)
				if actualType == agent.AgentType {
					return &pane
				}
			}
		}
	}

	return nil
}

// findAgentPaneByType finds an agent pane by type only (for established conversations)
func findAgentPaneByType(agentType string) *Pane {
	panes := getTmuxPanes()

	for _, pane := range panes {
		detectedType := detectAgentType(pane.Command)
		if detectedType == agentType {
			return &pane
		}

		// For node processes, check deeper
		if pane.Command == "node" {
			actualType := detectNodeAgent(pane.ID)
			if actualType == agentType {
				return &pane
			}
		}
	}

	return nil
}

func detectAgentType(command string) string {
	command = strings.ToLower(command)

	if strings.Contains(command, "claude") {
		return "claude"
	}
	if strings.Contains(command, "opencode") {
		return "opencode"
	}
	if strings.Contains(command, "coder") && !strings.Contains(command, "opencode") {
		return "coder"
	}
	if strings.Contains(command, "crush") {
		return "crush"
	}

	return "unknown"
}

func detectNodeAgent(paneID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Get pane PID
	cmd := exec.CommandContext(ctx, "tmux", "display-message", "-p", "-t", paneID, "#{pane_pid}")
	pidOutput, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	pid := strings.TrimSpace(string(pidOutput))
	if pid == "" {
		return "unknown"
	}

	// Get child processes
	cmd = exec.CommandContext(ctx, "pgrep", "-P", pid)
	childOutput, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	childPids := strings.Split(string(childOutput), "\n")
	for _, childPid := range childPids {
		childPid = strings.TrimSpace(childPid)
		if childPid == "" {
			continue
		}

		// Check child process command
		cmd = exec.CommandContext(ctx, "ps", "-p", childPid, "-o", "command=")
		cmdOutput, err := cmd.Output()
		if err != nil {
			continue
		}

		agentType := detectAgentType(string(cmdOutput))
		if agentType != "unknown" {
			return agentType
		}
	}

	return "unknown"
}

func sendMessage(paneID string, message string, targetAgent *RegistryEntry, registry []RegistryEntry) bool {
	// Format message with sender info and response instructions
	senderInfo := detectSenderFromRegistry(registry)
	formattedMessage := message

	if senderInfo != "" && senderInfo != "unknown" {
		// Add structured wrapper for receiving agent to parse
		// Include receiver name so they know who to respond to with --from flag
		formattedMessage = fmt.Sprintf(
			"{Receiving msg from: %s} \"%s\" {When ready to respond use: msg --from %s %s 'your return message'}",
			senderInfo, message, targetAgent.Name, senderInfo)
	}

	// Send message
	cmd := exec.Command("tmux", "send-keys", "-t", paneID, formattedMessage)
	if err := cmd.Run(); err != nil {
		return false
	}

	// Staggered Enter presses for reliability
	time.Sleep(100 * time.Millisecond)
	cmd = exec.Command("tmux", "send-keys", "-t", paneID, "C-m")
	cmd.Run()

	time.Sleep(100 * time.Millisecond)
	cmd = exec.Command("tmux", "send-keys", "-t", paneID, "C-m")
	cmd.Run()

	return true
}

func getCurrentPaneInfo(registry []RegistryEntry) string {
	// Get current pane's directory
	cmd := exec.Command("tmux", "display-message", "-p", "#{pane_current_path}")
	dirOutput, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	currentDir := strings.TrimSpace(string(dirOutput))

	// Get current pane's command
	cmd = exec.Command("tmux", "display-message", "-p", "#{pane_current_command}")
	cmdOutput, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	currentCmd := strings.TrimSpace(string(cmdOutput))

	// Detect agent type
	agentType := detectAgentType(currentCmd)
	if agentType == "unknown" && currentCmd == "node" {
		// For node processes, need deeper detection
		// Get the current pane ID
		cmd = exec.Command("tmux", "display-message", "-p", "#{session_name}:#{window_index}.#{pane_index}")
		paneOutput, _ := cmd.Output()
		paneID := strings.TrimSpace(string(paneOutput))
		if paneID != "" {
			// Try to detect what node process is actually running
			agentType = detectNodeAgent(paneID)
			// If still unknown, check if we're Claude (common case)
			if agentType == "unknown" {
				// Check if claude is running in this process tree
				pidCmd := exec.Command("pgrep", "-f", "claude")
				if pidOutput, err := pidCmd.Output(); err == nil && len(pidOutput) > 0 {
					agentType = "claude"
				}
			}
		}
	}

	// Find registered name for this agent
	for _, agent := range registry {
		if agent.AgentType == agentType && agent.Directory == currentDir {
			return agent.Name
		}
	}

	// Fallback to agent type if not registered
	if agentType != "unknown" {
		return agentType
	}

	return "unknown"
}

func detectSenderFromRegistry(registry []RegistryEntry) string {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}

	// Find agent by directory match
	for _, agent := range registry {
		if agent.Directory == currentDir {
			return agent.Name
		}
	}

	// If exact match not found, check if current dir is subdirectory of any registered agent
	for _, agent := range registry {
		if strings.HasPrefix(currentDir, agent.Directory) {
			return agent.Name
		}
	}

	return "unknown"
}

func showStatus() {
	fmt.Println("=== MESSAGING SYSTEM STATUS ===\n")

	// Load and show registry
	registry := loadRegistry()
	if registry != nil {
		fmt.Printf("Registered agents (%d):\n", len(registry))
		for _, agent := range registry {
			fmt.Printf("  - %s: %s @ %s", agent.Name, agent.AgentType, agent.Directory)

			// Check if active
			pane := findAgentPane(&agent)
			if pane != nil {
				fmt.Printf(" ✓ Active in %s\n", pane.ID)
			} else {
				fmt.Printf(" ✗ Not found\n")
			}
		}
	} else {
		fmt.Println("No registry found")
	}

	// Show active panes
	fmt.Println("\nActive tmux panes:")
	panes := getTmuxPanes()
	if len(panes) > 0 {
		for _, pane := range panes {
			agentType := detectAgentType(pane.Command)
			if pane.Command == "node" {
				detected := detectNodeAgent(pane.ID)
				if detected != "unknown" {
					agentType = detected
				}
			}

			// Only show AI agents
			if agentType != "unknown" {
				shortDir := pane.Directory
				if home := os.Getenv("HOME"); home != "" {
					shortDir = strings.Replace(shortDir, home, "~", 1)
				}
				// Truncate long paths
				if len(shortDir) > 40 {
					shortDir = "..." + shortDir[len(shortDir)-37:]
				}
				fmt.Printf("  %s: %s @ %s\n", pane.ID, agentType, shortDir)
			}
		}
	} else {
		fmt.Println("  No panes found")
	}
}