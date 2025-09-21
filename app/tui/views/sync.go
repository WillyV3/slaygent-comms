package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

// SyncViewData contains all data needed to render the sync customization view
type SyncViewData struct {
	// Text editing
	Editor       textarea.Model
	Modified     bool

	// State management
	Mode         SyncMode

	// UI components
	Help         help.Model

	// Terminal dimensions
	Width        int
	Height       int
}

type SyncMode int

const (
	ViewMode SyncMode = iota
	EditMode
	PreviewMode
)

// Default registry clause template - content only (markers added by script)
const DefaultRegistryClause = `# Inter-Agent Communication
@/Users/williamvansickleiii/.slaygent/registry.json

To send messages to other coding agents, use: ` + "`msg <agent_name> \"<message>\"`" + `
Example: ` + "`msg backend-dev \"Please update the API endpoint\"`" + `

IMPORTANT: When responding to messages, always use the --from flag:
` + "`msg --from <your_agent_name> <target_agent> \"<response>\"`" + `
This ensures proper conversation logging and tracking.

<!-- Registry automatically synced by slaygent-manager -->`

// KeyMap defines the key bindings for the sync view
type SyncKeyMap struct {
	Enter     key.Binding
	Escape    key.Binding
	Save      key.Binding
	Help      key.Binding
}

func NewSyncKeyMap() SyncKeyMap {
	return SyncKeyMap{
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "new line (in edit)"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back to agents"),
		),
		Save: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "custom sync (exit edit first)"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}
}

// ShortHelp returns key bindings for the short help view
func (k SyncKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Save, k.Escape}
}

// FullHelp returns key bindings for the full help view
func (k SyncKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Save, k.Help, k.Escape},
	}
}

// RenderSyncView renders the simplified sync customization view
func RenderSyncView(data SyncViewData) string {
	if data.Width == 0 || data.Height == 0 {
		return "Loading sync editor..."
	}

	// Style definitions
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87CEEB")).
		Bold(true).
		MarginBottom(1)

	// Build title and warning
	title := titleStyle.Render("CUSTOMIZE REGISTRY CLAUSE")

	// Calculate content area dimensions first
	contentHeight := data.Height - 16 // Reserve even more space for title, warning, and help
	contentWidth := data.Width - 4    // Side padding

	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B6B")).
		Italic(true).
		MarginBottom(1)

	warningText := "⚠️  WARNING: Modifying this updates all CLAUDE.md and AGENTS.md files on your system. If you change the file reference, agents won't have access to communication context. It's not recommended to change this unless you know what you're doing."
	warning := wrapToTerminal(warningStyle.Render(warningText), contentWidth)

	var content string
	switch data.Mode {
	case EditMode:
		content = renderSimpleEditMode(data, contentWidth, contentHeight)
	case PreviewMode:
		content = renderSimplePreviewMode(data, contentWidth, contentHeight)
	default:
		content = renderSimpleViewMode(data, contentWidth, contentHeight)
	}

	// Build help
	helpView := data.Help.View(NewSyncKeyMap())

	// Assemble the full view
	view := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
		title,
		warning,
		content,
		helpView)

	return wrapToTerminal(lipgloss.NewStyle().
		Padding(0, 1).
		Render(view), data.Width)
}


func renderSimpleEditMode(data SyncViewData, width, height int) string {
	// Style the editor
	editorStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#87CEEB"))

	// Add modification indicator
	modifiedIndicator := ""
	if data.Modified {
		modifiedIndicator = " " + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Render("●")
	}

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render(fmt.Sprintf("Editing Registry Clause%s - Press Tab to exit, then 'c' to custom sync", modifiedIndicator))

	editorView := editorStyle.Render(data.Editor.View())

	return fmt.Sprintf("%s\n%s", header, editorView)
}

func renderSimplePreviewMode(data SyncViewData, width, height int) string {
	// Style the preview
	previewStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00CED1")).
		Padding(1)

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("Preview: Registry Clause")

	// Show the raw content
	content := data.Editor.Value()
	if content == "" {
		content = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Render("(Empty clause)")
	}

	previewView := previewStyle.Render(content)

	return fmt.Sprintf("%s\n%s", header, previewView)
}

func renderSimpleViewMode(data SyncViewData, width, height int) string {
	// Show simple overview
	overviewStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#006666")).
		Padding(1)

	content := data.Editor.Value()
	preview := ""
	if content != "" {
		lines := strings.Split(content, "\n")
		if len(lines) > 0 {
			preview = strings.TrimSpace(lines[0])
			if len(preview) > 80 {
				preview = preview[:77] + "..."
			}
		}
		if len(lines) > 1 {
			preview += fmt.Sprintf("\n(%d lines total)", len(lines))
		}
	} else {
		preview = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Render("(Using default registry clause)")
	}

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("Tab to edit, 'c' to custom sync to all CLAUDE.md/AGENTS.md files")

	overviewView := overviewStyle.Render(preview)

	return fmt.Sprintf("%s\n%s", header, overviewView)
}

// BuildSyncEditor creates a new textarea for editing the registry clause
func BuildSyncEditor(width, height int) textarea.Model {
	editor := textarea.New()
	editor.Placeholder = "Enter registry clause content..."
	editor.CharLimit = 2000
	editor.SetWidth(width - 4)  // Account for border
	editor.SetHeight(height - 4) // Account for border
	editor.ShowLineNumbers = true
	editor.KeyMap.InsertNewline.SetEnabled(true)

	// Set the default content
	editor.SetValue(DefaultRegistryClause)

	// Styling
	editor.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("#2A2A2A"))
	editor.BlurredStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	return editor
}