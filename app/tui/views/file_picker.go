package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// DiscoveredFile represents a file found by discovery
type DiscoveredFile struct {
	Path      string
	Type      string
	Directory string
	Selected  bool
}

// RenderFilePicker renders the file picker interface
func RenderFilePicker(files []DiscoveredFile, selectedIndex int, loading bool, errorMsg string, spinners []spinner.Model, width, height int) string {
	if loading {
		return renderFilePickerLoading(spinners, width, height)
	}

	if errorMsg != "" {
		return renderFilePickerError(errorMsg, width, height)
	}

	if len(files) == 0 {
		return renderFilePickerEmpty(width, height)
	}

	return renderFilePickerList(files, selectedIndex, width, height)
}

// renderFilePickerLoading shows loading state while discovering files
func renderFilePickerLoading(spinners []spinner.Model, width, height int) string {
	// Create a line of all spinners
	var spinnerLine string
	if len(spinners) > 0 {
		for i, sp := range spinners {
			spinnerLine += sp.View()
			if i < len(spinners)-1 {
				spinnerLine += " "
			}
		}
	}

	content := fmt.Sprintf("%s\n\nDiscovering CLAUDE.md and AGENTS.md files...\n\nThis may take a moment for large file systems.", spinnerLine)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(width-2).
		Height(height-2).
		Align(lipgloss.Center, lipgloss.Center).
		Padding(1).
		Render(content)
}

// renderFilePickerError shows error state
func renderFilePickerError(errorMsg string, width, height int) string {
	var content string
	if strings.Contains(errorMsg, "fd command not found") {
		content = "fd command not found\n\n" +
			"Please install fd to use file picker:\n" +
			"macOS: brew install fd\n" +
			"Linux: apt install fd-find\n\n" +
			"Press ESC to return to sync view"
	} else {
		content = fmt.Sprintf("Error discovering files:\n\n%s\n\nPress ESC to return to sync view", errorMsg)
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Width(width-2).
		Height(height-2).
		Align(lipgloss.Center, lipgloss.Center).
		Padding(1).
		Render(content)
}

// renderFilePickerEmpty shows when no files are found
func renderFilePickerEmpty(width, height int) string {
	content := "No CLAUDE.md or AGENTS.md files found\n\n" +
		"Create CLAUDE.md files in your projects to use the sync feature.\n\n" +
		"Press ESC to return to sync view"

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("214")).
		Width(width-2).
		Height(height-2).
		Align(lipgloss.Center, lipgloss.Center).
		Padding(1).
		Render(content)
}

// renderFilePickerList renders the main file picker list
func renderFilePickerList(files []DiscoveredFile, selectedIndex int, width, height int) string {
	// Ensure minimum dimensions
	if width < 20 {
		width = 20
	}
	if height < 10 {
		height = 10
	}

	// Calculate available width for content (border + padding)
	contentWidth := width - 4 // 2 for border, 2 for padding

	// Header
	selectedCount := getSelectedCount(files)
	header := truncateText(fmt.Sprintf("Custom Sync - File Selection (Selected: %d/%d files)", selectedCount, len(files)), contentWidth)

	// Footer with instructions (responsive to width)
	var footer string
	if contentWidth >= 76 {
		footer = "[SPACE] Toggle • [A] Select All • [N] Select None • [F] Current Project\n" +
			"[ENTER] Sync Selected • [ESC] Cancel"
	} else if contentWidth >= 46 {
		footer = "[SPC] Toggle • [A] All • [N] None • [F] Project\n" +
			"[ENTER] Sync • [ESC] Cancel"
	} else {
		footer = "[SPC] Toggle • [A] All • [N] None\n" +
			"[ENTER] Sync • [ESC] Cancel"
	}

	// Calculate available height for file list
	headerLines := 1
	footerLines := strings.Count(footer, "\n") + 1 // Count actual footer lines
	spacingLines := 2 // blank lines between sections
	listHeight := height - 4 - headerLines - footerLines - spacingLines // 4 = border + padding

	// Ensure minimum list height
	if listHeight < 1 {
		listHeight = 1
	}

	// File list
	var fileLines []string
	if len(files) == 0 {
		fileLines = []string{"No files to display"}
	} else {
		// Calculate visible range for scrolling
		startIdx, endIdx := calculateVisibleRange(selectedIndex, len(files), listHeight)

		for i := startIdx; i < endIdx && i < len(files); i++ {
			file := files[i]
			line := renderFileListItem(file, i == selectedIndex, contentWidth)
			fileLines = append(fileLines, line)
		}

		// Fill remaining space to maintain consistent height
		for len(fileLines) < listHeight {
			fileLines = append(fileLines, strings.Repeat(" ", contentWidth))
		}
	}

	fileList := strings.Join(fileLines, "\n")

	// Combine all parts
	content := fmt.Sprintf("%s\n\n%s\n\n%s", header, fileList, footer)

	// Render with responsive styling
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(width-2).
		Height(height-2).
		Padding(1).
		Render(content)
}

// renderFileListItem renders a single file item in the list
func renderFileListItem(file DiscoveredFile, isSelected bool, maxWidth int) string {
	// Ensure minimum width
	if maxWidth < 10 {
		maxWidth = 10
	}

	// Checkbox
	checkbox := "[ ]"
	if file.Selected {
		checkbox = "[x]"
	}

	// Calculate available space for path (checkbox + spaces)
	pathSpace := maxWidth - 5 // " [ ] " = 5 chars

	// Convert to user-friendly display path and truncate
	displayPath := truncateText(makeDisplayPath(file.Path), pathSpace)

	// Create the line
	line := fmt.Sprintf(" %s %s", checkbox, displayPath)

	// Pad line to fill width
	if len(line) < maxWidth {
		line += strings.Repeat(" ", maxWidth-len(line))
	}

	// Style based on selection state
	style := lipgloss.NewStyle().Width(maxWidth)

	if isSelected {
		// Highlight current selection
		style = style.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
	} else if file.Selected {
		// Different color for selected files
		style = style.Foreground(lipgloss.Color("34"))
	}

	return style.Render(line)
}

// calculateVisibleRange determines which files should be visible in the scrollable list
func calculateVisibleRange(selectedIndex, totalFiles, visibleCount int) (int, int) {
	if totalFiles <= visibleCount {
		return 0, totalFiles
	}

	// Try to center the selected index
	start := selectedIndex - visibleCount/2
	if start < 0 {
		start = 0
	}

	end := start + visibleCount
	if end > totalFiles {
		end = totalFiles
		start = end - visibleCount
		if start < 0 {
			start = 0
		}
	}

	return start, end
}

// truncateText truncates text to fit within maxWidth
func truncateText(text string, maxWidth int) string {
	if len(text) <= maxWidth {
		return text
	}
	if maxWidth <= 3 {
		return "..."
	}
	return text[:maxWidth-3] + "..."
}

// getSelectedCount counts how many files are selected
func getSelectedCount(files []DiscoveredFile) int {
	count := 0
	for _, file := range files {
		if file.Selected {
			count++
		}
	}
	return count
}

// makeDisplayPath converts absolute paths to user-friendly display paths
func makeDisplayPath(absolutePath string) string {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return absolutePath // Fallback to absolute path if we can't get home
	}

	// Convert to relative path from home directory
	if strings.HasPrefix(absolutePath, homeDir) {
		relPath, err := filepath.Rel(homeDir, absolutePath)
		if err == nil {
			return "~/" + relPath
		}
	}

	return absolutePath // Fallback to absolute path
}