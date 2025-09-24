package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DiscoveredFile represents a CLAUDE.md or AGENTS.md file found by fd
type DiscoveredFile struct {
	Path      string // Full path to the file
	Type      string // "CLAUDE.md" or "AGENTS.md"
	Directory string // Parent directory name for display
	Selected  bool   // Whether user has selected this file
}

// fileDiscoveryMsg contains the result of file discovery
type fileDiscoveryMsg struct {
	files []DiscoveredFile
	error string
}

// fileDiscoveryTickMsg for loading animation
type fileDiscoveryTickMsg struct{}

// discoverFiles finds all CLAUDE.md and AGENTS.md files using fd command
func discoverFiles() ([]DiscoveredFile, error) {
	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use fd to find all CLAUDE.md and AGENTS.md files from home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "fd", "-t", "f", "-H", "^(CLAUDE|AGENTS)\\.md$", homeDir)

	output, err := cmd.Output()
	if err != nil {
		// Check if fd command is not found
		if strings.Contains(err.Error(), "executable file not found") {
			return nil, &fdNotFoundError{}
		}
		return nil, err
	}

	// Parse output into DiscoveredFile structs
	var files []DiscoveredFile
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Determine file type
		fileName := filepath.Base(line)
		if fileName != "CLAUDE.md" && fileName != "AGENTS.md" {
			continue // Skip if not exactly our target files
		}

		// Get directory name for display
		dir := filepath.Dir(line)
		dirName := filepath.Base(dir)
		if dirName == "." {
			dirName = "/"
		}

		files = append(files, DiscoveredFile{
			Path:      line,
			Type:      fileName,
			Directory: dirName,
			Selected:  false, // Default to unselected
		})
	}

	return files, nil
}

// fdNotFoundError represents when fd command is not available
type fdNotFoundError struct{}

func (e *fdNotFoundError) Error() string {
	return "fd command not found - install with: brew install fd"
}

// selectCurrentProjectFiles automatically selects files in/under current working directory
func selectCurrentProjectFiles(files []DiscoveredFile) []DiscoveredFile {
	cwd, err := os.Getwd()
	if err != nil {
		return files // Return unchanged if we can't get cwd
	}

	// Auto-select files that are in or under the current directory
	for i := range files {
		if strings.HasPrefix(files[i].Path, cwd) {
			files[i].Selected = true
		}
	}

	return files
}

// getSelectedFiles returns only the files that are currently selected
func getSelectedFiles(files []DiscoveredFile) []DiscoveredFile {
	var selected []DiscoveredFile
	for _, file := range files {
		if file.Selected {
			selected = append(selected, file)
		}
	}
	return selected
}

// getSelectedCount returns the number of selected files
func getSelectedCount(files []DiscoveredFile) int {
	count := 0
	for _, file := range files {
		if file.Selected {
			count++
		}
	}
	return count
}

// toggleFileSelection toggles the selection state of a file at given index
func toggleFileSelection(files []DiscoveredFile, index int) []DiscoveredFile {
	if index >= 0 && index < len(files) {
		files[index].Selected = !files[index].Selected
	}
	return files
}

// selectAllFiles selects all files in the list
func selectAllFiles(files []DiscoveredFile) []DiscoveredFile {
	for i := range files {
		files[i].Selected = true
	}
	return files
}

// deselectAllFiles deselects all files in the list
func deselectAllFiles(files []DiscoveredFile) []DiscoveredFile {
	for i := range files {
		files[i].Selected = false
	}
	return files
}