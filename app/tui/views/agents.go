
package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	lipglosstable "github.com/charmbracelet/lipgloss/table"
	"github.com/evertras/bubble-table/table"
)

// SSHConnection represents a connection to a remote machine
type SSHConnection struct {
	Name           string `json:"name"`
	SSHKey         string `json:"ssh_key"`
	ConnectCommand string `json:"connect_command"`
}

// AgentsViewData contains all data needed to render the agents view
type AgentsViewData struct {
	Table         table.Model  // Changed to bubble-table Model
	Rows          [][]string
	Registry      interface{ GetName(string, string) string }
	SSHConnCount  int  // Number of SSH connections
	InputMode     bool
	InputBuffer   string
	InputTarget   string  // What we're inputting for
	TempSSHName   string  // Temporary SSH name during registration
	TempSSHKey    string  // Temporary SSH key during registration
	SyncConfirm   bool
	Syncing       bool
	SyncMessage   string
	Progress      progress.Model
	Width         int
}

// RenderAgentsView renders the agents view
func RenderAgentsView(data AgentsViewData) string {
	// ASCII title art with simple 3-color gradient
	topStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#87CEEB")) // Light blue

	middleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF6B6B")) // Coral

	bottomStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ECDC4")) // Turquoise

	title := strings.Join([]string{
		topStyle.Render(" ▄▄ ▝▜                       ▗"),
		topStyle.Render("▐▘ ▘ ▐   ▄▖ ▗ ▗  ▄▄  ▄▖ ▗▗▖ ▗▟▄ "),
		middleStyle.Render("▝▙▄  ▐  ▝ ▐ ▝▖▞ ▐▘▜ ▐▘▐ ▐▘▐  ▐  "),
		middleStyle.Render("  ▝▌ ▐  ▗▀▜  ▙▌ ▐ ▐ ▐▀▀ ▐ ▐  ▐ "),
		bottomStyle.Render("▝▄▟▘ ▝▄ ▝▄▜  ▜  ▝▙▜ ▝▙▞ ▐ ▐  ▝▄ "),
		bottomStyle.Render("             ▞   ▖▐            "),
		bottomStyle.Render("            ▝▘   ▝▘         "),
}, "\n")


// SSH Connection Status
var connectionStatus string
if data.SSHConnCount > 0 {
	connectionStatus = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87CEEB")).
		Bold(true).
		Render(fmt.Sprintf("🌐 %d SSH machine%s connected", data.SSHConnCount, func() string {
			if data.SSHConnCount == 1 { return "" }
			return "s"
		}()))
} else {
	connectionStatus = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("🌐 No SSH machines connected")
}

// Controls with grey styling
controlsStyle := lipgloss.NewStyle().
	Foreground(lipgloss.Color("#888888")).
	MarginTop(1)

controls := controlsStyle.Render(strings.Join([]string{
	"Getting around this page:",
	"↑/↓ or j/k: Navigate",
	"a: Register/unregister agent",
	"z: Register SSH connection",
	"x: Manage SSH connections",
	"r: Refresh agent list",
	"s: Sync agents/claude.md",
	"e: Edit injected sync content",
	"m: View Message History",
	"?: Learn how to use Slaygent",
	"q or Ctrl+C: Quit",
}, "\n"))

// Use Lipgloss JoinHorizontal for proper side-by-side layout
header := lipgloss.JoinHorizontal(
	lipgloss.Top,    // Align to top
	lipgloss.JoinVertical(lipgloss.Left, title, "", connectionStatus), // Left side: ASCII art + connection status
	"        ",      // More spacing between columns
	controls,        // Right side: controls
)

// Table title
tableTitle := lipgloss.NewStyle().
	Foreground(lipgloss.Color("#87CEEB")).
	Bold(true).
	Align(lipgloss.Center).
	Render("Use This Page To Register and Unregister AI Coding Tools in TMUX")

// Table subtitle (footer note) - only show when not in input mode
tableSubtitle := ""
if !data.InputMode {
	tableSubtitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D4AC0D")).
		Align(lipgloss.Center).
		Render("* Registering an Agent adds it to the registry and makes it available for inter-agent communication")
}

view := "\n" + header + "\n\n" + tableTitle + "\n\n" + data.Table.View() + "\n\n" + tableSubtitle + "\n"

// Show sync confirmation prompt
if data.SyncConfirm {
	fullView := view + "\nSync registry to all CLAUDE.md files? (y/N): "
	return wrapToTerminal(fullView, data.Width)
}

// Show sync progress or success message
if data.Syncing {
	syncingText := lipgloss.NewStyle().Foreground(lipgloss.Color("#00CED1")).Render("Syncing CLAUDE.md files...")
	progressView := "\n" + data.Progress.View() + "\n" + syncingText
	fullView := view + progressView
	return wrapToTerminal(fullView, data.Width)
}

// Show sync success message
if data.SyncMessage != "" {
	fullView := view + "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render(data.SyncMessage)
	return wrapToTerminal(fullView, data.Width)
}

// Show input prompt if in input mode
if data.InputMode {
	darkPinkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#C71585")).Bold(true)

	switch data.InputTarget {
	case "register":
		// Agent registration prompt
		selectedRow := data.Table.GetHighlightedRowIndex()
		if selectedRow >= 0 && selectedRow < len(data.Rows) {
			row := data.Rows[selectedRow]
			agentType := row[2]
			fullDirectory := row[1]  // Full path for registry
			displayDirectory := filepath.Base(fullDirectory)  // Short name for display
			registerText := fmt.Sprintf("Register %s in %s", agentType, displayDirectory)
			prompt := "\n" + darkPinkStyle.Render(registerText) + fmt.Sprintf("\n\nName: %s_", data.InputBuffer)
			fullView := view + prompt + "\n\nPress Enter to save, Esc to cancel\n"
			return wrapToTerminal(fullView, data.Width)
		}

	case "ssh-name":
		// SSH machine name prompt
		registerText := "Register SSH Connection - Step 1/3"
		prompt := "\n" + darkPinkStyle.Render(registerText) + fmt.Sprintf("\n\nMachine name: %s_", data.InputBuffer)
		fullView := view + prompt + "\n\nPress Enter to continue, Esc to cancel\n"
		return wrapToTerminal(fullView, data.Width)

	case "ssh-key-picker":
		// This case should not be reached since we handle the file picker in main View()
		// But included for completeness
		registerText := fmt.Sprintf("Register SSH Connection '%s' - Step 2/3: Selecting SSH Key", data.TempSSHName)
		prompt := "\n" + darkPinkStyle.Render(registerText) + "\n\nFile picker is active..."
		fullView := view + prompt + "\n"
		return wrapToTerminal(fullView, data.Width)

	case "ssh-command":
		// SSH connect command prompt
		registerText := fmt.Sprintf("Register SSH Connection '%s' - Step 3/3", data.TempSSHName)
		keyText := ""
		if data.TempSSHKey != "" {
			keyFileName := filepath.Base(data.TempSSHKey)
			keyText = fmt.Sprintf(" (Key: %s)", keyFileName)
		}
		prompt := "\n" + darkPinkStyle.Render(registerText + keyText) + fmt.Sprintf("\n\nConnect command: %s_", data.InputBuffer)
		fullView := view + prompt + "\n\nPress Enter to save, Esc to cancel\n"
		return wrapToTerminal(fullView, data.Width)
	}
}

// Show error message if no tmux server
if len(data.Rows) > 0 && data.Rows[0][0] == "ERROR" {
	view += "\n⚠️  No tmux sessions found. Run 'tmux new' to start.\n"
}

// Show selected row info
selectedRowIndex := data.Table.GetHighlightedRowIndex()
if len(data.Rows) > 0 && selectedRowIndex >= 0 && selectedRowIndex < len(data.Rows) && data.Rows[0][0] != "ERROR" {
	selectedRow := data.Rows[selectedRowIndex]
	agentType := selectedRow[2]
	fullDirectory := selectedRow[1]  // data.Rows still has full path

	// Show registered name if exists
	status := ""
	brownStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8B4513")) // Brown color
	if data.Registry != nil {
		if name := data.Registry.GetName(agentType, fullDirectory); name != "" {
			status = brownStyle.Render(fmt.Sprintf("\nSelected: %s [%s]", selectedRow[3], name))
		} else {
			status = brownStyle.Render(fmt.Sprintf("\nSelected: %s (%s)", selectedRow[3], agentType))
		}
	}
	view += status
}

// Wrap entire view to terminal width
fullView := view + "\n"
return wrapToTerminal(fullView, data.Width)
}

// wrapToTerminal wraps content to terminal width if available
func wrapToTerminal(content string, width int) string {
	if width > 0 {
		return lipgloss.NewStyle().
			Width(width).
			MaxWidth(width).
			Render(content)
	}
	return content
}

// BuildTableWithSelection creates a table with the current selection highlighted (legacy lipgloss table)
func BuildTableWithSelection(rows [][]string, selected int, registry interface{ GetName(string, string) string }) *lipglosstable.Table {
	re := lipgloss.NewRenderer(os.Stdout)
	baseStyle := re.NewStyle().Padding(0, 1)
	selectedStyle := baseStyle.Foreground(lipgloss.Color("#01BE85")).Background(lipgloss.Color("#00432F"))
	agentColors := map[string]lipgloss.Color{
		"claude":   lipgloss.Color("#CC5500"), // Burnt orange
		"gemini":   lipgloss.Color("#7B68EE"), // Purply blue
		"coder":    lipgloss.Color("#00FF00"), // Green
		"codex":    lipgloss.Color("#008B8B"), // Teal
		"opencode": lipgloss.Color("#FFFF00"), // Bright yellow
		"crush":    lipgloss.Color("#FF87D7"),
		"unknown":  lipgloss.Color("#929292"),
	}
	headers := []string{"PANE", "DIRECTORY", "AGENT", "NAME", "STATUS", "REGISTERED"}

	// Show only last folder name in directory column
	displayRows := make([][]string, len(rows))
	for i, row := range rows {
		displayRow := make([]string, len(row))
		copy(displayRow, row)

		// Truncate directory to last folder name (column 1)
		if len(displayRow) > 1 && displayRow[1] != "" {
			displayRow[1] = filepath.Base(displayRow[1])
		}

		displayRows[i] = displayRow
	}

	return lipglosstable.New().
		Headers(headers...).
		Rows(displayRows...).
		Border(lipgloss.NormalBorder()).
		BorderStyle(re.NewStyle().Foreground(lipgloss.Color("#006666"))). // Darker teal border
		StyleFunc(func(row, col int) lipgloss.Style {
			// The table component handles headers separately
			// All rows passed to StyleFunc are data rows, starting from 0
			if row < 0 || row >= len(displayRows) {
				return baseStyle
			}

			// Highlight selected row - now correctly matches selected
			if row == selected {
				return selectedStyle
			}

			even := row%2 == 0

			switch col {
			case 1: // DIRECTORY column - unique color per directory
				if row >= len(rows) || col >= len(rows[row]) {
					return baseStyle
				}
				// Generate unique color using ANSI 256 colors (more compatible)
				// Use colors 21-231 which are the color cube (avoid grayscale)
				colorNum := 21 + (row * 30) % 210
				return baseStyle.Foreground(lipgloss.Color(fmt.Sprintf("%d", colorNum)))
			case 0: // PANE column - purple styling
				if row >= len(rows) || col >= len(rows[row]) {
					return baseStyle
				}
				return baseStyle.Foreground(lipgloss.Color("#9B59B6")) // Purple for pane numbers
			case 2: // AGENT column
				if row >= len(rows) || col >= len(rows[row]) {
					return baseStyle
				}

				// Always use agent type colors (don't change to blue when registered)
				color, ok := agentColors[rows[row][col]]
				if !ok {
					return baseStyle
				}
				return baseStyle.Foreground(color)
			case 3: // NAME column - style registered names in bold blue
				if row >= len(rows) {
					return baseStyle
				}

				// Check if this row has a registered agent (checkmark in column 5)
				if len(rows[row]) > 5 && rows[row][5] == "✓" {
					// Agent is registered, apply blue styling to the name
					return baseStyle.Foreground(lipgloss.Color("#5DADE2")).Bold(true)
				}
				// Not registered (shows "NR"), use default styling
				return baseStyle
			case 5: // REGISTERED column - green checkmarks, red x's
				if row >= len(rows) || col >= len(rows[row]) {
					return baseStyle
				}

				value := rows[row][col]
				if value == "✓" {
					return baseStyle.Foreground(lipgloss.Color("#00FF00")) // Green for registered
				} else if value == "✗" {
					return baseStyle.Foreground(lipgloss.Color("#FF0000")) // Red for not registered
				}
				return baseStyle
			}

			if even {
				return baseStyle.Foreground(lipgloss.Color("245"))
			}
			return baseStyle.Foreground(lipgloss.Color("252"))
		}).
		Border(lipgloss.ThickBorder())
}

// Column keys for bubble-table
const (
	columnKeyPane       = "pane"
	columnKeyDirectory  = "directory"
	columnKeyAgent      = "agent"
	columnKeyName       = "name"
	columnKeyStatus     = "status"
	columnKeyMachine    = "machine"
	columnKeyRegistered = "registered"
)

// BuildBubbleTable creates a new bubble-table with flex columns and multiline support
func BuildBubbleTable(rows [][]string, registry interface{ GetName(string, string) string }, width int) table.Model {
	// Define columns with flex capabilities for better responsive layout
	columns := []table.Column{
		table.NewFlexColumn(columnKeyPane, "PANE", 2).WithStyle(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#9B59B6")).Align(lipgloss.Center)),
		table.NewFlexColumn(columnKeyDirectory, "DIRECTORY", 3).WithStyle(
			lipgloss.NewStyle().Align(lipgloss.Left)),
		table.NewColumn(columnKeyAgent, "AGENT", 8).WithStyle(
			lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewFlexColumn(columnKeyName, "NAME", 3).WithStyle(
			lipgloss.NewStyle().Align(lipgloss.Left)),
		table.NewColumn(columnKeyStatus, "STATUS", 7).WithStyle(
			lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn(columnKeyMachine, "MACHINE", 8).WithStyle(
			lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn(columnKeyRegistered, "REGISTERED?", 12).WithStyle(
			lipgloss.NewStyle().Align(lipgloss.Center)),
	}

	// Agent colors map
	agentColors := map[string]lipgloss.Color{
		"claude":   lipgloss.Color("#CC5500"), // Burnt orange
		"gemini":   lipgloss.Color("#7B68EE"), // Purply blue
		"coder":    lipgloss.Color("#00FF00"), // Green
		"codex":    lipgloss.Color("#008B8B"), // Teal
		"opencode": lipgloss.Color("#FFFF00"), // Bright yellow
		"crush":    lipgloss.Color("#FF87D7"),
		"unknown":  lipgloss.Color("#929292"),
	}

	// Convert rows to bubble-table Row format
	tableRows := make([]table.Row, 0, len(rows))
	for i, row := range rows {
		if len(row) < 7 {
			continue // Skip incomplete rows (now expecting 7 columns)
		}

		// Truncate directory to last folder name
		directory := row[1]
		if directory != "" {
			directory = filepath.Base(directory)
		}

		// Create row data
		rowData := table.RowData{
			columnKeyPane:       row[0],
			columnKeyDirectory:  directory,
			columnKeyAgent:      row[2],
			columnKeyName:       row[3],
			columnKeyStatus:     row[4],
			columnKeyMachine:    row[5],
			columnKeyRegistered: row[6],
		}

		// Apply agent-specific styling to the AGENT column
		if agentColor, ok := agentColors[row[2]]; ok {
			agentCell := table.NewStyledCell(row[2], lipgloss.NewStyle().
				Foreground(agentColor).Align(lipgloss.Center))
			rowData[columnKeyAgent] = agentCell
		}

		// Style registered names in bold blue
		if len(row) > 6 && row[6] == "✓" {
			// Override name cell styling for registered agents
			nameCell := table.NewStyledCell(row[3], lipgloss.NewStyle().
				Foreground(lipgloss.Color("#5DADE2")).Bold(true))
			rowData[columnKeyName] = nameCell
		}

		// Style machine column with distinct colors
		machineColor := lipgloss.Color("#87CEEB") // Default baby blue for "host"
		if row[5] != "host" {
			// Use different color for remote machines
			machineColor = lipgloss.Color("#FFB347") // Orange for remote machines
		}
		machineCell := table.NewStyledCell(row[5], lipgloss.NewStyle().
			Foreground(machineColor).Align(lipgloss.Center))
		rowData[columnKeyMachine] = machineCell

		// Style registered column with colors and manual centering
		if row[6] == "✓" {
			regCell := table.NewStyledCell("     ✓     ", lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00")))
			rowData[columnKeyRegistered] = regCell
		} else if row[6] == "✗" {
			regCell := table.NewStyledCell("     ✗     ", lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000")))
			rowData[columnKeyRegistered] = regCell
		}

		// Generate unique directory colors
		colorNum := 21 + (i * 30) % 210
		dirCell := table.NewStyledCell(directory, lipgloss.NewStyle().
			Foreground(lipgloss.Color(fmt.Sprintf("%d", colorNum))))
		rowData[columnKeyDirectory] = dirCell

		// Create final table row with all styled cells
		tableRow := table.NewRow(rowData)
		tableRows = append(tableRows, tableRow)
	}

	// Create and configure the table with flex and multiline support
	bubbleTable := table.New(columns).
		WithRows(tableRows).
		HeaderStyle(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87CEEB")).
			Bold(true).
			Align(lipgloss.Center)).
		SelectableRows(false).
		Focused(true).
		WithMultiline(true).
		WithTargetWidth(width).
		WithBaseStyle(lipgloss.NewStyle().
			BorderForeground(lipgloss.Color("#006666"))).
		HighlightStyle(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87CEEB")).
			Background(lipgloss.Color("#1E3A5F")).
			Bold(true))

	return bubbleTable
}
