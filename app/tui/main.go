package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/evertras/bubble-table/table"
	"slaygent-manager/history"
	"slaygent-manager/views"
)

type model struct {
	table       table.Model  // Changed to bubble-table Model
	rows        [][]string
	registry    *Registry
	inputMode   bool   // Are we in input mode?
	inputBuffer string // What the user is typing
	inputTarget string // What we're inputting for (e.g., "register", "sync")
	syncConfirm bool   // Are we in sync confirmation mode?
	syncing     bool   // Are we currently syncing?
	syncMessage string // Message to show after sync completes
	progress    progress.Model // Progress bar for syncing
	viewMode    string // "agents", "messages", "sync", or "help"
	historyModel *history.Model
	messagesViewport viewport.Model
	messagesFocus string // "conversations" or "messages" - which panel has focus
	selectedMessage int // Selected message index when in messages panel
	deleteConfirm bool // Are we in delete confirmation mode?
	deleteTarget int   // Which conversation ID to delete

	// Sync customization fields
	syncEditor       textarea.Model
	syncMode         views.SyncMode
	syncModified     bool
	syncHelp         help.Model

	// Help view
	helpModel *views.HelpModel

	width       int // Terminal width
	height      int // Terminal height
}

func (m model) Init() tea.Cmd {
	// Set window title and disable auto-refresh to prevent duplication
	return tea.SetWindowTitle("Slaygent Manager")
}

// initializeSyncComponents sets up the sync customization components
func (m model) initializeSyncComponents() model {
	if m.syncHelp.Width == 0 { // Check if already initialized
		m.syncEditor = views.BuildSyncEditor(
			m.width-12, // Account for padding and borders
			m.height-20, // Account for title, warning, and help - keep consistent
		)
		m.syncMode = views.ViewMode
		m.syncHelp = help.New()
	}
	return m
}


type refreshMsg struct{}
type syncCompleteMsg struct{
	filesUpdated int
}
type syncProgressMsg struct{
	current int
	total   int
	fileName string
}
type syncTickMsg time.Time
type resetProgressMsg struct{}


func (m model) View() string {
	// Show help view if active
	if m.viewMode == "help" {
		if m.helpModel != nil {
			return m.helpModel.View()
		}
		return "Help not available"
	}

	// Show sync view if active
	if m.viewMode == "sync" {
		return views.RenderSyncView(views.SyncViewData{
			Editor:   m.syncEditor,
			Mode:     m.syncMode,
			Modified: m.syncModified,
			Help:     m.syncHelp,
			Width:    m.width,
			Height:   m.height,
		})
	}

	// Show messages view if active
	if m.viewMode == "messages" {
		return views.RenderMessagesView(views.MessagesViewData{
			HistoryModel:     m.historyModel,
			MessagesViewport: m.messagesViewport,
			MessagesFocus:    m.messagesFocus,
			SelectedMessage:  m.selectedMessage,
			DeleteConfirm:    m.deleteConfirm,
			DeleteTarget:     m.deleteTarget,
			Width:            m.width,
			Height:           m.height,
		})
	}

	// Show agents view
	return views.RenderAgentsView(views.AgentsViewData{
		Table:         m.table,
		Rows:          m.rows,
		Registry:      m.registry,
		InputMode:     m.inputMode,
		InputBuffer:   m.inputBuffer,
		SyncConfirm:   m.syncConfirm,
		Syncing:       m.syncing,
		SyncMessage:   m.syncMessage,
		Progress:      m.progress,
		Width:         m.width,
	})
}

// findSyncScript returns the path to the sync script, checking multiple locations
func findSyncScript(scriptName string) string {
	// Try relative path (development)
	relativePath := "../scripts/" + scriptName
	if _, err := os.Stat(relativePath); err == nil {
		return relativePath
	}

	// Try Homebrew libexec locations for different platforms
	possiblePaths := []string{
		"/opt/homebrew/lib/slaygent-comms/" + scriptName,              // macOS ARM Homebrew
		"/usr/local/lib/slaygent-comms/" + scriptName,                 // macOS Intel Homebrew
		"/home/linuxbrew/.linuxbrew/lib/slaygent-comms/" + scriptName, // Linux Homebrew
		"/usr/lib/slaygent-comms/" + scriptName,                       // System install
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Fallback to relative path (will fail but with clear error)
	return relativePath
}

// runSyncCommand executes the sync script
func (m model) runSyncCommand() tea.Cmd {
	return func() tea.Msg {
		// Find and execute sync script
		scriptPath := findSyncScript("sync-claude.sh")
		cmd := exec.Command("bash", "-c", fmt.Sprintf("echo 'y' | %s", scriptPath))
		cmd.Dir = "."
		output, err := cmd.Output()
		if err != nil {
			return syncCompleteMsg{filesUpdated: 0}
		}

		// Count how many files were updated by looking for "✓ Synced" in output
		lines := strings.Split(string(output), "\n")
		filesUpdated := 0
		for _, line := range lines {
			if strings.Contains(line, "✓ Synced") {
				filesUpdated++
			}
		}

		return syncCompleteMsg{filesUpdated: filesUpdated}
	}
}

// runCustomSyncCommand executes the custom sync script with user's content
func (m model) runCustomSyncCommand() tea.Cmd {
	return func() tea.Msg {
		// Get the custom content from the editor
		customContent := m.syncEditor.Value()

		// Find custom sync script and create heredoc command
		scriptPath := findSyncScript("custom-sync-claude.sh")
		scriptCmd := fmt.Sprintf(`echo 'y' | %s "$(cat <<'EOF'
%s
EOF
)"`, scriptPath, customContent)

		// Execute custom sync script with the content via heredoc
		cmd := exec.Command("bash", "-c", scriptCmd)
		cmd.Dir = "."
		output, err := cmd.Output()
		if err != nil {
			return syncCompleteMsg{filesUpdated: 0}
		}

		// Count how many files were updated by looking for "✓ Synced" in output
		lines := strings.Split(string(output), "\n")
		filesUpdated := 0
		for _, line := range lines {
			if strings.Contains(line, "✓ Synced") {
				filesUpdated++
			}
		}

		return syncCompleteMsg{filesUpdated: filesUpdated}
	}
}

// syncTickCmd creates a tick for progress animation
func syncTickCmd() tea.Cmd {
	return tea.Tick(800*time.Millisecond, func(t time.Time) tea.Msg {
		return syncTickMsg(t)
	})
}


// refreshAll refreshes tmux data, syncs registry, and rebuilds table
func (m model) refreshAll() model {
	// Get fresh tmux data
	rows, err := getTmuxPanes(m.registry)
	if err != nil {
		m.rows = [][]string{
			{"ERROR", "No tmux server", "unknown", "tmux-error", "error", "✗"},
			{"", "Run 'tmux new' to start", "", "", "", ""},
		}
	} else {
		m.rows = rows
		// Sync registry to remove stale entries
		if m.registry != nil {
			m.registry.SyncWithActive(rows)
		}
	}

	// Rebuild table with bubble-table
	m.table = views.BuildBubbleTable(m.rows, m.registry, m.width)
	return m
}


func main() {
	// Initialize registry
	registry, err := NewRegistry()
	if err != nil {
		fmt.Printf("Warning: Failed to initialize registry: %v\n", err)
		// Continue without registry
		registry = nil
	}

	// Get  tmux data
	rows, err := getTmuxPanes(registry)
	if err != nil {
		// Show error state with helpful message
		rows = [][]string{
			{"ERROR", "No tmux server running", "unknown", "tmux-error", "error", "✗"},
			{"HELP", "Run 'tmux new' to start", "", "", "", ""},
		}
	}

	// Handle empty result (no AI agents found)
	if len(rows) == 0 {
		rows = [][]string{
			{"INFO", "No AI agents detected", "unknown", "scan-result", "idle", "✗"},
			{"HELP", "Start claude/opencode/coder/crush", "", "", "", ""},
		}
	}

	// Initialize progress bar
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 60

	// Initialize history model
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".slaygent", "messages.db")
	historyModel, err := history.New(dbPath)
	if err != nil {
		// Continue without history - Messages view will show "Database unavailable"
		historyModel = nil
	} else {
		// Load initial conversations
		historyModel.LoadConversations()
	}

	// Initialize viewport for messages
	vp := viewport.New(80, 20)

	m := model{
		rows:     rows,
		registry: registry,
		progress: prog,
		viewMode: "agents",
		historyModel: historyModel,
		messagesViewport: vp,
		width:    120,  // Default width, will be updated by WindowSizeMsg
		height:   30,   // Default height, will be updated by WindowSizeMsg
	}
	m.table = views.BuildBubbleTable(m.rows, m.registry, m.width)
	defer func() {
		if m.historyModel != nil {
			m.historyModel.Close()
		}
	}()

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
