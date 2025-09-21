package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// getTmuxPanes returns tmux pane information as a slice of row data
func getTmuxPanes(registry *Registry) ([][]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Check if tmux server is running
	if !isTmuxRunning(ctx) {
		return nil, fmt.Errorf("tmux server is not running")
	}

	// Get pane information using tmux list-panes
	// Include both session_name and session_id to handle named sessions properly
	format := "#{session_name}:#{session_id}:#{window_index}.#{pane_index}:#{pane_current_path}:#{pane_current_command}:#{?pane_active,active,idle}"
	cmd := exec.CommandContext(ctx, "tmux", "list-panes", "-a", "-F", format)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get tmux panes: %w", err)
	}

	rows, err := parseTmuxOutput(string(output))
	if err != nil {
		return nil, err
	}

	// Update registration status and name for each row
	for i, row := range rows {
		if len(row) >= 6 {
			agentType := row[2]  // AGENT column
			directory := row[1]  // DIRECTORY column
			if registry != nil {
				if registry.IsRegistered(agentType, directory) {
					rows[i][5] = "✓"  // Update REGISTERED column
					// Replace NAME column with registered name
					if name := registry.GetName(agentType, directory); name != "" {
						rows[i][3] = name  // Update NAME column with registered name
					}
				} else {
					rows[i][5] = "✗"  // Update REGISTERED column to not registered
					rows[i][3] = "NR"  // Not Registered
				}
			}
		}
	}

	return rows, nil
}

// isTmuxRunning checks if tmux server is accessible
func isTmuxRunning(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "tmux", "has-session")
	err := cmd.Run()
	return err == nil
}

// parseTmuxOutput parses tmux list-panes output into display rows
func parseTmuxOutput(output string) ([][]string, error) {
	if strings.TrimSpace(output) == "" {
		return [][]string{}, nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var rows [][]string

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 6 {
			continue // Skip malformed lines (now expecting 6 parts)
		}

		sessionName := parts[0]      // session name (like "go-0" or "0" if unnamed)
		_ = parts[1]                 // session ID (like "$23") - not needed for targeting
		windowPane := parts[2]       // window.pane format (like "1.1")
		directory := parts[3]        // current path
		command := parts[4]          // current command
		status := parts[5]           // active/idle

		// Use session name for pane targeting (works with both named and unnamed sessions)
		// This is what tmux expects when targeting panes
		fullPaneID := sessionName + ":" + windowPane

		// Detect AI agent type - check direct command first
		agentType := detectAgentType(command)

		// For node processes, always check what's actually running
		if command == "node" {
			agentType = detectAgentInPane(fullPaneID)
		}

		// Skip non-AI agents (only show claude, opencode, coder, crush)
		if agentType == "unknown" {
			continue
		}

		// Check registration status using real registry
		registered := "✗"
		// We'll check registration after we have the model with registry

		// Create display name using session name for better readability
		displayName := sessionName + ":" + windowPane

		rows = append(rows, []string{
			fullPaneID,     // Use session_name:window.pane for tmux targeting
			directory,      // Full directory path
			agentType,
			displayName,    // Display session_name:window.pane
			status,
			registered,     // Will be updated later with registry check
		})
	}

	return rows, nil
}

// detectAgentType analyzes a tmux pane command to determine AI agent type
func detectAgentType(command string) string {
	command = strings.ToLower(command)

	// Direct command detection
	if strings.Contains(command, "claude") || strings.Contains(command, "claude-code") {
		return "claude"
	}
	if strings.Contains(command, "opencode") || strings.Contains(command, "open-code") {
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

// detectAgentInPane checks for AI agent by examining the process running in the pane
func detectAgentInPane(paneID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Get the PID of the process in this specific pane using display-message
	// This ensures we get only one PID for the exact pane
	pidCmd := exec.CommandContext(ctx, "tmux", "display-message", "-p", "-t", paneID, "#{pane_pid}")
	pidOutput, err := pidCmd.Output()
	if err != nil {
		return "unknown"
	}

	pid := strings.TrimSpace(string(pidOutput))
	if pid == "" {
		return "unknown"
	}

	// Get child processes of this PID (the shell's children)
	pgrepCmd := exec.CommandContext(ctx, "pgrep", "-P", pid)
	childPids, err := pgrepCmd.Output()
	if err != nil {
		return "unknown"
	}

	// Check each child process
	for _, childPid := range strings.Split(string(childPids), "\n") {
		childPid = strings.TrimSpace(childPid)
		if childPid == "" {
			continue
		}

		agentType := checkProcessCommand(ctx, childPid)
		if agentType != "unknown" {
			return agentType
		}
	}

	return "unknown"
}

// checkProcessCommand checks a single process to determine if it's an AI agent
func checkProcessCommand(ctx context.Context, pid string) string {
	// Get the command for this PID
	psCmd := exec.CommandContext(ctx, "ps", "-p", pid, "-o", "command=")
	cmdOutput, err := psCmd.Output()
	if err != nil {
		return "unknown"
	}

	command := strings.ToLower(strings.TrimSpace(string(cmdOutput)))

	// Check for AI agent commands (either direct or as arguments to node/python)
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