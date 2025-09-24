package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// RenderSyncProgress renders the sync progress interface with spinner and logs
func RenderSyncProgress(title string, logs []string, sp spinner.Model, active bool, errorMsg string, width, height int) string {
	// Ensure minimum dimensions
	if width < 30 {
		width = 30
	}
	if height < 10 {
		height = 10
	}

	if errorMsg != "" {
		return renderSyncProgressError(errorMsg, width, height)
	}

	if !active && len(logs) == 0 {
		// Initial state before sync starts
		return renderSyncProgressInitial(title, width, height)
	}

	return renderSyncProgressActive(title, logs, sp, active, width, height)
}

// renderSyncProgressError shows error state
func renderSyncProgressError(errorMsg string, width, height int) string {
	content := fmt.Sprintf("Sync Error:\n\n%s\n\nPress ESC to return", errorMsg)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Width(width-2).
		Height(height-2).
		Align(lipgloss.Center, lipgloss.Center).
		Padding(1).
		Render(content)
}

// renderSyncProgressInitial shows initial state
func renderSyncProgressInitial(title string, width, height int) string {
	content := fmt.Sprintf("%s\n\nPreparing to sync selected files...\n\nPress ESC to cancel", title)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(width-2).
		Height(height-2).
		Align(lipgloss.Center, lipgloss.Center).
		Padding(1).
		Render(content)
}

// renderSyncProgressActive shows spinner and logs
func renderSyncProgressActive(title string, logs []string, sp spinner.Model, active bool, width, height int) string {
	// Header with spinner
	var header string
	if active {
		header = fmt.Sprintf("%s %s", sp.View(), title)
	} else {
		header = fmt.Sprintf("%s - Complete!", title)
	}

	// Calculate available height for logs
	borderAndPadding := 4 // 2 for border, 2 for padding
	headerLines := 1
	footerLines := 2 // "Press ESC to return" + spacing
	spacingLines := 2 // blank lines between sections
	logsHeight := height - borderAndPadding - headerLines - footerLines - spacingLines

	// Ensure minimum logs height
	if logsHeight < 3 {
		logsHeight = 3
	}

	// Prepare logs display
	var logLines []string
	if len(logs) == 0 {
		logLines = []string{"Starting sync process..."}
	} else {
		// Show recent logs (scroll to bottom)
		startIdx := 0
		if len(logs) > logsHeight {
			startIdx = len(logs) - logsHeight
		}

		for i := startIdx; i < len(logs); i++ {
			// Truncate long log lines to fit width
			logLine := truncateLogLine(logs[i], width-6)
			logLines = append(logLines, logLine)
		}
	}

	// Fill remaining space if needed
	for len(logLines) < logsHeight {
		logLines = append(logLines, "")
	}

	logsDisplay := strings.Join(logLines, "\n")

	// Footer
	var footer string
	if active {
		footer = "Press ESC to cancel sync"
	} else {
		footer = "Press ESC to return to file picker"
	}

	// Combine all parts
	content := fmt.Sprintf("%s\n\n%s\n\n%s", header, logsDisplay, footer)

	// Render with responsive styling
	var borderColor string
	if active {
		borderColor = "62" // Blue while active
	} else {
		borderColor = "34" // Green when complete
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Width(width-2).
		Height(height-2).
		Padding(1).
		Render(content)
}

// truncateLogLine truncates log lines to fit within maxWidth
func truncateLogLine(line string, maxWidth int) string {
	if len(line) <= maxWidth {
		return line
	}
	if maxWidth <= 3 {
		return "..."
	}
	return line[:maxWidth-3] + "..."
}