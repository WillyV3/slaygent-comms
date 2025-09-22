package views

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SSHConnectionsViewData contains all data needed to render the SSH connections view
type SSHConnectionsViewData struct {
	Connections      []SSHConnection
	SelectedIndex    int
	DeleteConfirm    bool
	DeleteTarget     int
	Width            int
	Height           int
}

// Styling constants
var (
	sshTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87CEEB")).
		Bold(true)

	sshControlsStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	sshSelectedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#87CEEB")).
		Foreground(lipgloss.Color("#000000")).
		Bold(true)

	sshNormalStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	sshDeleteStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B6B")).
		Bold(true)
)

// RenderSSHConnectionsView renders the SSH connections management view
func RenderSSHConnectionsView(data SSHConnectionsViewData) string {
	if data.Width == 0 || data.Height == 0 {
		panic("SSH connections view dimensions not initialized")
	}

	// Build header
	title := sshTitleStyle.Render("SSH Connections")

	// Build connections list
	connectionsList := renderConnectionsList(data)

	// Build controls
	controls := sshControlsStyle.Render("↑/↓: navigate • d: delete connection • ESC: back to agents")

	// Delete confirmation prompt
	var deletePrompt string
	if data.DeleteConfirm && data.DeleteTarget < len(data.Connections) {
		targetName := data.Connections[data.DeleteTarget].Name
		deletePrompt = "\n" + sshDeleteStyle.Render(fmt.Sprintf("Delete connection '%s'? (y/n)", targetName))
	}

	return fmt.Sprintf("\n%s\n\n%s%s\n\n%s", title, connectionsList, deletePrompt, controls)
}

// renderConnectionsList builds the list of SSH connections
func renderConnectionsList(data SSHConnectionsViewData) string {
	if len(data.Connections) == 0 {
		return sshControlsStyle.Render("No SSH connections configured.\nPress 'z' in agents view to add connections.")
	}

	var lines []string

	for i, conn := range data.Connections {
		// Format connection details
		keyName := filepath.Base(conn.SSHKey)
		if keyName == "" {
			keyName = "No key specified"
		}

		// Truncate long commands for display
		command := conn.ConnectCommand
		if len(command) > 50 {
			command = command[:47] + "..."
		}

		line := fmt.Sprintf("%-20s │ %-20s │ %s",
			conn.Name,
			keyName,
			command,
		)

		// Apply styling based on selection
		if i == data.SelectedIndex {
			line = sshSelectedStyle.Render("> " + line)
		} else {
			line = sshNormalStyle.Render("  " + line)
		}

		lines = append(lines, line)
	}

	// Add header
	header := sshControlsStyle.Render("  Name                 │ SSH Key              │ Connect Command")
	separator := sshControlsStyle.Render("  " + strings.Repeat("─", 70))

	return fmt.Sprintf("%s\n%s\n%s", header, separator, strings.Join(lines, "\n"))
}