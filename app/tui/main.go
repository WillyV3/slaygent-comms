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
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"slaygent-manager/history"
	"slaygent-manager/views"
)

type model struct {
	table       table.Model  // Changed to bubble-table Model
	rows        [][]string
	registry    *Registry
	sshRegistry *SSHRegistry
	inputMode   bool   // Are we in input mode?
	inputBuffer string // What the user is typing
	inputTarget string // What we're inputting for (e.g., "register", "ssh-name", "ssh-key", "ssh-key-picker", "ssh-command")
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

	// SSH connection being built
	tempSSHName    string
	tempSSHKey     string
	tempSSHCommand string

	// SSH key selection
	sshKeys         []string
	selectedSSHKey  int

	// SSH connections view
	sshSelectedIndex int
	sshDeleteConfirm bool
	sshDeleteTarget  int

	// File picker for custom sync
	filePickerMode     bool
	discoveredFiles    []DiscoveredFile
	filePickerIndex    int
	filePickerLoading  bool
	filePickerError    string
	filePickerSpinners []spinner.Model // Multiple spinners for fun!

	// Sync progress
	syncProgressMode    bool
	syncProgressTitle   string
	syncProgressLogs    []string
	syncProgressActive  bool
	syncProgressError   string
	syncProgressSpinner spinner.Model

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

// getSSHKeys returns a list of SSH key files from ~/.ssh directory
func getSSHKeys() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}

	sshDir := filepath.Join(home, ".ssh")
	files, err := os.ReadDir(sshDir)
	if err != nil {
		return []string{}
	}

	var keys []string
	for _, file := range files {
		if !file.IsDir() {
			name := file.Name()
			// Include only private SSH keys (exclude .pub files and other non-key files)
			if !strings.HasSuffix(name, ".pub") &&  // Exclude public keys
			   !strings.HasSuffix(name, ".old") &&  // Exclude backup files
			   name != "config" &&                  // Exclude SSH config
			   name != "known_hosts" &&             // Exclude known hosts
			   name != "authorized_keys" &&         // Exclude authorized keys
			   (strings.HasSuffix(name, ".pem") ||  // Include .pem private keys
			    strings.HasSuffix(name, ".key") ||  // Include .key private keys
			    !strings.Contains(name, ".")) {     // Include keys without extensions (common for SSH)
				keys = append(keys, filepath.Join(sshDir, name))
			}
		}
	}
	return keys
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

	// Show file picker if active (takes precedence over sync view)
	if m.filePickerMode {
		// Convert to views.DiscoveredFile slice
		var viewFiles []views.DiscoveredFile
		for _, f := range m.discoveredFiles {
			viewFiles = append(viewFiles, views.DiscoveredFile{
				Path:      f.Path,
				Type:      f.Type,
				Directory: f.Directory,
				Selected:  f.Selected,
			})
		}
		return views.RenderFilePicker(
			viewFiles,
			m.filePickerIndex,
			m.filePickerLoading,
			m.filePickerError,
			m.filePickerSpinners,
			m.width,
			m.height,
		)
	}

	// Show sync progress if active (takes precedence over sync view)
	if m.syncProgressMode {
		return views.RenderSyncProgress(
			m.syncProgressTitle,
			m.syncProgressLogs,
			m.syncProgressSpinner,
			m.syncProgressActive,
			m.syncProgressError,
			m.width,
			m.height,
		)
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

	// Show SSH connections view if active
	if m.viewMode == "ssh_connections" {
		connections := []views.SSHConnection{}
		if m.sshRegistry != nil {
			for _, conn := range m.sshRegistry.GetConnections() {
				connections = append(connections, views.SSHConnection{
					Name:           conn.Name,
					SSHKey:         conn.SSHKey,
					ConnectCommand: conn.ConnectCommand,
				})
			}
		}

		return views.RenderSSHConnectionsView(views.SSHConnectionsViewData{
			Connections:   connections,
			SelectedIndex: m.sshSelectedIndex,
			DeleteConfirm: m.sshDeleteConfirm,
			DeleteTarget:  m.sshDeleteTarget,
			Width:         m.width,
			Height:        m.height,
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

	// Show SSH key selector if active
	if m.inputTarget == "ssh-key-picker" {
		title := fmt.Sprintf("Select SSH Key for '%s'", m.tempSSHName)
		instructions := "↑/↓: navigate • Enter: select • Esc: cancel"

		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87CEEB")).
			Bold(true).
			Margin(1, 0)

		instructionsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Margin(0, 0, 1, 0)

		content := titleStyle.Render(title) + "\n" +
			instructionsStyle.Render(instructions) + "\n"

		if len(m.sshKeys) == 0 {
			content += lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF6B6B")).
				Render("No SSH keys found in ~/.ssh directory")
		} else {
			for i, key := range m.sshKeys {
				keyName := filepath.Base(key)
				if i == m.selectedSSHKey {
					content += lipgloss.NewStyle().
						Background(lipgloss.Color("#87CEEB")).
						Foreground(lipgloss.Color("#000000")).
						Render("> " + keyName) + "\n"
				} else {
					content += "  " + keyName + "\n"
				}
			}
		}

		return content
	}

	// Show agents view
	sshConnCount := 0
	if m.sshRegistry != nil {
		sshConnCount = len(m.sshRegistry.GetConnections())
	}

	return views.RenderAgentsView(views.AgentsViewData{
		Table:         m.table,
		Rows:          m.rows,
		Registry:      m.registry,
		SSHConnCount:  sshConnCount,
		InputMode:     m.inputMode,
		InputBuffer:   m.inputBuffer,
		InputTarget:   m.inputTarget,
		TempSSHName:   m.tempSSHName,
		TempSSHKey:    m.tempSSHKey,
		Syncing:       m.syncing,
		SyncMessage:   m.syncMessage,
		Progress:      m.progress,
		Width:         m.width,
	})
}

// findSyncScript returns the path to the sync script, checking multiple locations
func findSyncScript(scriptName string) string {
	// PRIORITY 1: Dynamic Homebrew detection (works on any machine)
	if brewPrefix := getHomebrewPrefix(); brewPrefix != "" {
		// Check lib location FIRST (stable, version-independent)
		libPath := filepath.Join(brewPrefix, "lib", "slaygent-comms", scriptName)
		if _, err := os.Stat(libPath); err == nil {
			return libPath
		}

		// Check Cellar as fallback (for older versions)
		cellarBase := filepath.Join(brewPrefix, "Cellar", "slaygent-comms")
		if entries, err := os.ReadDir(cellarBase); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					dynamicPath := filepath.Join(cellarBase, entry.Name(), "libexec", scriptName)
					if _, err := os.Stat(dynamicPath); err == nil {
						return dynamicPath
					}
				}
			}
		}
	}

	// PRIORITY 2: Standard Homebrew locations (fallback)
	standardPaths := []string{
		"/opt/homebrew/Cellar/slaygent-comms",      // macOS ARM
		"/usr/local/Cellar/slaygent-comms",         // macOS Intel
		"/home/linuxbrew/.linuxbrew/Cellar/slaygent-comms", // Linux
	}

	for _, cellarBase := range standardPaths {
		if entries, err := os.ReadDir(cellarBase); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					dynamicPath := filepath.Join(cellarBase, entry.Name(), "libexec", scriptName)
					if _, err := os.Stat(dynamicPath); err == nil {
						return dynamicPath
					}
				}
			}
		}
	}

	// PRIORITY 3: Development mode (relative path)
	relativePath := "../scripts/" + scriptName
	if _, err := os.Stat(relativePath); err == nil {
		return relativePath
	}

	// PRIORITY 4: System install locations
	systemPaths := []string{
		"/opt/homebrew/lib/slaygent-comms/" + scriptName,
		"/usr/local/lib/slaygent-comms/" + scriptName,
		"/home/linuxbrew/.linuxbrew/lib/slaygent-comms/" + scriptName,
		"/usr/lib/slaygent-comms/" + scriptName,
	}

	for _, path := range systemPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// FALLBACK: Return path that will cause clear error
	return "/usr/bin/false" // This will fail with clear error message
}

// getHomebrewPrefix returns the Homebrew prefix if brew is available
func getHomebrewPrefix() string {
	cmd := exec.Command("brew", "--prefix")
	cmd.Env = os.Environ() // Ensure full environment is available
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// discoverFilesCommand starts the file discovery process
func (m model) discoverFilesCommand() tea.Cmd {
	return tea.Batch(
		// Start the spinner animation
		m.startFileDiscoverySpinner(),
		// Start the actual file discovery
		func() tea.Msg {
			files, err := discoverFiles()
			if err != nil {
				return fileDiscoveryMsg{error: err.Error()}
			}

			// Auto-select current project files
			files = selectCurrentProjectFiles(files)

			return fileDiscoveryMsg{files: files}
		},
	)
}

// startFileDiscoverySpinner starts a spinner animation during file discovery
func (m model) startFileDiscoverySpinner() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return fileDiscoveryTickMsg{}
	})
}

// runCustomSyncOnSelectedFiles executes custom sync on user-selected files
func (m model) runCustomSyncOnSelectedFiles() tea.Cmd {
	return func() tea.Msg {
		selectedFiles := getSelectedFiles(m.discoveredFiles)
		if len(selectedFiles) == 0 {
			return syncCompleteMsg{filesUpdated: 0}
		}

		customContent := m.syncEditor.Value()
		if strings.TrimSpace(customContent) == "" {
			return syncCompleteMsg{filesUpdated: 0}
		}

		filesUpdated := 0
		for _, file := range selectedFiles {
			if err := updateFileWithCustomContent(file.Path, customContent); err == nil {
				filesUpdated++
			}
		}

		return syncCompleteMsg{filesUpdated: filesUpdated}
	}
}

// updateFileWithCustomContent updates a single file with custom sync content
func updateFileWithCustomContent(filePath, customContent string) error {
	// Read existing file content
	existingContent, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Create backup
	backupPath := filePath + ".backup"
	if err := os.WriteFile(backupPath, existingContent, 0644); err != nil {
		return err
	}

	// Markers for sync content
	startMarker := "<!-- SLAYGENT-REGISTRY-START -->"
	endMarker := "<!-- SLAYGENT-REGISTRY-END -->"

	content := string(existingContent)

	// Check if markers exist
	startIdx := strings.Index(content, startMarker)
	endIdx := strings.Index(content, endMarker)

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		// Replace existing content between markers
		before := content[:startIdx]
		after := content[endIdx+len(endMarker):]
		newContent := before + startMarker + "\n" + customContent + "\n" + endMarker + after
		return os.WriteFile(filePath, []byte(newContent), 0644)
	} else {
		// Append new content with markers
		newContent := content + "\n\n" + startMarker + "\n" + customContent + "\n" + endMarker + "\n"
		return os.WriteFile(filePath, []byte(newContent), 0644)
	}
}

// runSyncCommand executes the sync script
func (m model) runSyncCommand() tea.Cmd {
	return func() tea.Msg {
		// Find and execute sync script
		scriptPath := findSyncScript("sync-claude.sh")
		cmd := exec.Command("bash", "-c", fmt.Sprintf("echo 'y' | %s", scriptPath))
		cmd.Dir = os.Getenv("HOME")
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

// runSyncProgressCommand executes sync for selected files with progress updates
func (m model) runSyncProgressCommand(selectedFiles []DiscoveredFile) tea.Cmd {
	return func() tea.Msg {
		customContent := m.syncEditor.Value()
		if strings.TrimSpace(customContent) == "" {
			return syncProgressErrorMsg{error: "No custom content to sync"}
		}

		// Send initial log
		go func() {
			// This would normally be sent as a message, but for simplicity we'll use a channel or similar
		}()

		totalFiles := len(selectedFiles)
		successCount := 0

		for i, file := range selectedFiles {
			// Write content to the file
			if err := writeFileContent(file.Path, customContent); err != nil {
				// Log error (in a real implementation, we'd send progress messages here)
				_ = fmt.Sprintf("[%d/%d] Failed to sync %s: %v", i+1, totalFiles, file.Path, err)
			} else {
				// Log success
				_ = fmt.Sprintf("[%d/%d] Successfully synced %s", i+1, totalFiles, file.Path)
				successCount++
			}
		}

		return syncProgressCompleteMsg{
			filesUpdated: successCount,
			totalFiles:   totalFiles,
		}
	}
}

// Message types for sync progress
type syncProgressLogMsg struct {
	log string
}

type syncProgressCompleteMsg struct {
	filesUpdated int
	totalFiles   int
}

type syncProgressErrorMsg struct {
	error string
}

// writeFileContent writes custom content to the specified file
func writeFileContent(filePath, content string) error {
	// Read existing file
	existingBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	existingContent := string(existingBytes)

	// Find registry section markers
	startMarker := "<!-- SLAYGENT-REGISTRY-START -->"
	endMarker := "<!-- SLAYGENT-REGISTRY-END -->"

	startIdx := strings.Index(existingContent, startMarker)
	endIdx := strings.Index(existingContent, endMarker)

	if startIdx == -1 || endIdx == -1 {
		// No registry section found, append content
		newContent := existingContent + "\n\n" + content + "\n"
		return os.WriteFile(filePath, []byte(newContent), 0644)
	}

	// Replace content between markers
	before := existingContent[:startIdx]
	after := existingContent[endIdx+len(endMarker):]
	newContent := before + startMarker + "\n" + content + "\n" + endMarker + after

	return os.WriteFile(filePath, []byte(newContent), 0644)
}

// syncTickCmd creates a tick for progress animation
func syncTickCmd() tea.Cmd {
	return tea.Tick(800*time.Millisecond, func(t time.Time) tea.Msg {
		return syncTickMsg(t)
	})
}


// refreshAll refreshes tmux data, syncs registry, and rebuilds table
func (m model) refreshAll() model {
	// Reload SSH registry to pick up changes
	if sshRegistry, err := NewSSHRegistry(); err == nil {
		m.sshRegistry = sshRegistry
	}

	// Get fresh tmux data from local and remote machines
	rows, err := getTmuxPanesWithSSH(m.registry, m.sshRegistry)
	if err != nil {
		m.rows = [][]string{
			{"ERROR", "No tmux server", "unknown", "tmux-error", "error", "host", "✗"},
			{"", "Run 'tmux new' to start", "", "", "", "", ""},
		}
	} else {
		m.rows = rows
		// No auto-adoption - remote agents are display-only and cannot be registered locally
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

	// Initialize SSH registry
	sshRegistry, err := NewSSHRegistry()
	if err != nil {
		fmt.Printf("Warning: Failed to initialize SSH registry: %v\n", err)
		// Continue without SSH registry
		sshRegistry = nil
	}

	// Get tmux data from local and remote machines
	rows, err := getTmuxPanesWithSSH(registry, sshRegistry)
	if err != nil {
		// Show error state with helpful message
		rows = [][]string{
			{"ERROR", "No tmux server running", "unknown", "tmux-error", "error", "host", "✗"},
			{"HELP", "Run 'tmux new' to start", "", "", "", "", ""},
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
		rows:        rows,
		registry:    registry,
		sshRegistry: sshRegistry,
		progress:    prog,
		viewMode:    "agents",
		historyModel: historyModel,
		messagesViewport: vp,
		width:       120,  // Default width, will be updated by WindowSizeMsg
		height:      30,   // Default height, will be updated by WindowSizeMsg
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
