package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"slaygent-manager/views"
)

// updateMessagesViewport centralizes how we update the messages viewport
// When focus is "conversations" or selectedMessage is -1: show all messages normally (faint)
// When focus is "messages" and selectedMessage >= 0: show with selection highlighting
func (m *model) updateMessagesViewport() {
	if m.historyModel == nil {
		return
	}

	var content string
	// If focus is on conversations panel OR no message selected, show normal view
	if m.messagesFocus == "conversations" || m.selectedMessage < 0 {
		content = m.historyModel.FormatMessages()  // All messages faint, no highlighting
	} else {
		// Focus is on messages panel AND a message is selected
		content = m.historyModel.FormatMessagesWithSelection(m.selectedMessage)
	}

	// Wrap content to viewport width
	wrappedContent := lipgloss.NewStyle().
		Width(m.messagesViewport.Width).
		Render(content)
	m.messagesViewport.SetContent(wrappedContent)

	// Scroll to keep selected message in view when navigating
	if m.messagesFocus == "messages" && m.selectedMessage >= 0 {
		// Count lines to find where the selected message is
		lines := strings.Split(wrappedContent, "\n")
		if m.selectedMessage < len(lines) {
			// Calculate position - try to center the selected message
			targetLine := m.selectedMessage
			viewportHeight := m.messagesViewport.Height

			// Calculate the line to scroll to (center the selected message if possible)
			scrollTo := targetLine - (viewportHeight / 2)
			if scrollTo < 0 {
				scrollTo = 0
			}

			// Set the viewport position using the proper method
			m.messagesViewport.SetYOffset(scrollTo)
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 4
		// Update help model dimensions if it exists
		if m.helpModel != nil {
			m.helpModel.Update(msg.Width, msg.Height)
		}

		// Update viewport dimensions for messages view
		if m.viewMode == "messages" {
			// Match the calculation from View()
			availableWidth := m.width - 4
			leftPanelWidth := availableWidth / 3
			if leftPanelWidth < 25 {
				leftPanelWidth = 25
			}
			rightPanelWidth := availableWidth - leftPanelWidth - 2
			panelHeight := m.height - 8

			// Viewport is inside the right panel, account for borders
			m.messagesViewport.Width = rightPanelWidth - 4
			m.messagesViewport.Height = panelHeight - 4

			// Re-render messages with new width if we have content
			if m.historyModel != nil && m.historyModel.GetSelectedConversation() != nil {
				m.updateMessagesViewport()
			}
		}

		// Rebuild table with new width for flex columns
		m.table = views.BuildBubbleTable(m.rows, m.registry, m.width)


		return m, nil
	case syncTickMsg:
		if m.syncing && m.progress.Percent() < 1.0 {
			cmd := m.progress.IncrPercent(0.1)
			return m, tea.Batch(syncTickCmd(), cmd)
		}
		return m, nil
	case syncCompleteMsg:
		m.progress.SetPercent(1.0) // Complete at 100%
		m.syncing = false
		m.syncMessage = fmt.Sprintf("✓ Successfully updated %d CLAUDE.md files with registry context", msg.filesUpdated)
		// Reset progress and message after a brief delay
		return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
			return resetProgressMsg{}
		})
	case resetProgressMsg:
		m.progress.SetPercent(0)
		m.syncMessage = "" // Clear the success message
		return m, nil
	case syncProgressLogMsg:
		// Add log to the sync progress logs
		m.syncProgressLogs = append(m.syncProgressLogs, msg.log)
		return m, nil
	case syncProgressCompleteMsg:
		// Sync is complete
		m.syncProgressActive = false
		completionMsg := fmt.Sprintf("Successfully synced %d out of %d files", msg.filesUpdated, msg.totalFiles)
		m.syncProgressLogs = append(m.syncProgressLogs, completionMsg)
		return m, nil
	case syncProgressCompleteWithLogsMsg:
		// Sync is complete with full logs
		m.syncProgressActive = false
		m.syncProgressLogs = msg.logs // Replace with all collected logs
		finalMsg := fmt.Sprintf("Sync complete: %d out of %d files updated successfully", msg.filesUpdated, msg.totalFiles)
		m.syncProgressLogs = append(m.syncProgressLogs, finalMsg)
		// Note: Keep spinner running to show completion state, it will be cleaned up on ESC
		return m, nil
	case syncProgressErrorMsg:
		// Sync failed
		m.syncProgressActive = false
		m.syncProgressError = msg.error
		return m, nil
	case spinner.TickMsg:
		if m.syncProgressMode && m.syncProgressActive {
			var cmd tea.Cmd
			m.syncProgressSpinner, cmd = m.syncProgressSpinner.Update(msg)
			return m, cmd
		} else if m.filePickerMode && m.filePickerLoading {
			var cmds []tea.Cmd
			for i := range m.filePickerSpinners {
				var cmd tea.Cmd
				m.filePickerSpinners[i], cmd = m.filePickerSpinners[i].Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			return m, tea.Batch(cmds...)
		}
		return m, nil
	case progress.FrameMsg:
		if m.syncing {
			progressModel, cmd := m.progress.Update(msg)
			m.progress = progressModel.(progress.Model)
			return m, cmd
		}
		return m, nil
	case fileDiscoveryMsg:
		m.filePickerLoading = false
		if msg.error != "" {
			m.filePickerError = msg.error
		} else {
			m.discoveredFiles = msg.files
			m.filePickerIndex = 0
			m.filePickerError = ""
		}
		return m, nil
	case fileDiscoveryTickMsg:
		// Just for loading animation - no action needed
		return m, nil
	case refreshMsg:
		// Auto-refresh disabled to prevent duplication
		// Use manual refresh with 'r' key only
	case tea.KeyMsg:
		// Sync confirmation removed - only use 'e' key for sync customization

		// Handle SSH key selection mode
		if m.inputTarget == "ssh-key-picker" {
			switch msg.String() {
			case "up":
				if m.selectedSSHKey > 0 {
					m.selectedSSHKey--
				}
			case "down":
				if m.selectedSSHKey < len(m.sshKeys)-1 {
					m.selectedSSHKey++
				}
			case "enter":
				// Select the current SSH key and move to command input
				if len(m.sshKeys) > 0 && m.selectedSSHKey < len(m.sshKeys) {
					m.tempSSHKey = m.sshKeys[m.selectedSSHKey]
					m.inputMode = true
					m.inputBuffer = ""
					m.inputTarget = "ssh-command"
				}
			case "esc":
				// Cancel SSH registration
				m.inputMode = false
				m.inputTarget = ""
				m.tempSSHName = ""
				m.tempSSHKey = ""
				m.tempSSHCommand = ""
			}
			return m, nil
		}

		// Handle input mode first
		if m.inputMode {
			switch msg.String() {
			case "enter":
				// Handle different input targets
				switch m.inputTarget {
				case "register":
					// Save agent registration with the entered name (only for local agents)
					selectedRowIndex := m.table.GetHighlightedRowIndex()
					if m.inputBuffer != "" && selectedRowIndex >= 0 && selectedRowIndex < len(m.rows) {
						row := m.rows[selectedRowIndex]
						if len(row) >= 7 {  // Make sure we have machine column
							agentType := row[2]     // AGENT column
							fullDirectory := row[1] // DIRECTORY column (full path)
							machine := row[5]       // MACHINE column
							// Only allow registration of local agents (machine == "host")
							if machine == "host" {
								m.registry.RegisterWithMachine(m.inputBuffer, agentType, fullDirectory, machine)
							}
						}
					}
					// Exit input mode
					m.inputMode = false
					m.inputBuffer = ""
					m.inputTarget = ""
					// Refresh everything
					m = m.refreshAll()

				case "ssh-name":
					// Save machine name and move to SSH key picker
					if m.inputBuffer != "" {
						m.tempSSHName = m.inputBuffer
						m.inputBuffer = ""
						m.inputTarget = "ssh-key-picker"
						// Load SSH keys
						m.sshKeys = getSSHKeys()
						m.selectedSSHKey = 0
						m.inputMode = false  // No text input for key selection
					}

				case "ssh-key-picker":
					// SSH key selection completed, move to command input
					if len(m.sshKeys) > 0 && m.selectedSSHKey < len(m.sshKeys) {
						m.tempSSHKey = m.sshKeys[m.selectedSSHKey]
					}
					m.inputMode = true
					m.inputBuffer = ""
					m.inputTarget = "ssh-command"

				case "ssh-command":
					// Save SSH connection and exit input mode
					if m.inputBuffer != "" {
						m.tempSSHCommand = m.inputBuffer
						// Save the complete SSH connection
						if m.sshRegistry != nil {
							m.sshRegistry.AddConnection(m.tempSSHName, m.tempSSHKey, m.tempSSHCommand)
							// Refresh agents table to show new remote agents
							m = m.refreshAll()
						}
						// Clear temp fields
						m.tempSSHName = ""
						m.tempSSHKey = ""
						m.tempSSHCommand = ""
					}
					// Exit input mode
					m.inputMode = false
					m.inputBuffer = ""
					m.inputTarget = ""
				}
			case "esc":
				// Cancel input mode and clear temp SSH fields
				m.inputMode = false
				m.inputBuffer = ""
				m.inputTarget = ""
				m.tempSSHName = ""
				m.tempSSHKey = ""
				m.tempSSHCommand = ""
			case "backspace", "delete":
				if len(m.inputBuffer) > 0 {
					m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
				}
			default:
				// Add character to buffer
				if len(msg.String()) == 1 {
					m.inputBuffer += msg.String()
				}
			}
			return m, nil
		}

		// Handle file picker mode
		if m.filePickerMode {
			switch msg.String() {
			case "esc":
				// Clean exit from file picker mode
				m.filePickerMode = false
				m.discoveredFiles = nil
				m.filePickerIndex = 0
				m.filePickerLoading = false
				m.filePickerError = ""
				// Reset all spinners to stop any pending ticks
				m.filePickerSpinners = nil
				return m, nil
			case "up", "k":
				if len(m.discoveredFiles) > 0 && m.filePickerIndex > 0 {
					m.filePickerIndex--
				}
				return m, nil
			case "down", "j":
				if len(m.discoveredFiles) > 0 && m.filePickerIndex < len(m.discoveredFiles)-1 {
					m.filePickerIndex++
				}
				return m, nil
			case " ": // Space to toggle selection
				if len(m.discoveredFiles) > 0 && m.filePickerIndex < len(m.discoveredFiles) {
					m.discoveredFiles = toggleFileSelection(m.discoveredFiles, m.filePickerIndex)
				}
				return m, nil
			case "a", "A": // Select all
				m.discoveredFiles = selectAllFiles(m.discoveredFiles)
				return m, nil
			case "n", "N": // Select none
				m.discoveredFiles = deselectAllFiles(m.discoveredFiles)
				return m, nil
			case "f", "F": // Select current project files
				cwd, _ := os.Getwd()
				for i := range m.discoveredFiles {
					m.discoveredFiles[i].Selected = strings.HasPrefix(m.discoveredFiles[i].Path, cwd)
				}
				return m, nil
			case "enter":
				// Execute sync on selected files
				selectedCount := getSelectedCount(m.discoveredFiles)
				if selectedCount > 0 {
					// Get selected files for sync
					selectedFiles := getSelectedFiles(m.discoveredFiles)

					// Exit file picker mode and start sync progress
					m.filePickerMode = false
					m.syncProgressMode = true
					m.syncProgressTitle = fmt.Sprintf("Syncing %d files", selectedCount)
					m.syncProgressLogs = []string{}
					m.syncProgressActive = true
					m.syncProgressError = ""

					// Initialize spinner
					s := spinner.New()
					s.Spinner = spinner.Dot
					s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
					m.syncProgressSpinner = s

					return m, tea.Batch(m.syncProgressSpinner.Tick, m.runSyncProgressCommand(selectedFiles))
				}
				return m, nil
			}
			return m, nil
		}

		// Handle sync progress mode
		if m.syncProgressMode {
			switch msg.String() {
			case "esc":
				// Clean exit from sync progress mode
				m.syncProgressMode = false
				m.syncProgressActive = false
				m.syncProgressLogs = nil
				m.syncProgressError = ""
				// Reset spinner to stop any pending ticks
				m.syncProgressSpinner = spinner.Model{}

				// Return to file picker if files are still available
				if len(m.discoveredFiles) > 0 {
					m.filePickerMode = true
				} else {
					// Go back to sync view if no files available
					m.viewMode = "sync"
				}
				return m, nil
			case "q", "ctrl+c":
				// Allow quit from sync progress
				return m, tea.Quit
			}
			// In sync progress mode, ignore other key inputs
			return m, nil
		}

		// Normal mode key handling
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "m":
			// Toggle to messages view
			if m.viewMode == "agents" {
				m.viewMode = "messages"
				m.messagesFocus = "conversations" // Default focus to conversations panel

				// Set up viewport dimensions for messages view (same calculation as WindowSizeMsg)
				availableWidth := m.width - 4
				leftPanelWidth := availableWidth / 3
				if leftPanelWidth < 25 {
					leftPanelWidth = 25
				}
				rightPanelWidth := availableWidth - leftPanelWidth - 2
				panelHeight := m.height - 8

				// Configure viewport dimensions (inside right panel, account for borders)
				m.messagesViewport.Width = rightPanelWidth - 4
				m.messagesViewport.Height = panelHeight - 4

				if m.historyModel != nil {
					m.historyModel.LoadConversations()
					// Load messages for first conversation if available
					if m.historyModel.HasConversations() {
						m.historyModel.SelectedConv = 0
						m.selectedMessage = -1  // Reset message selection when switching to messages view (-1 = no selection)
						conv := m.historyModel.GetSelectedConversation()
						if conv != nil {
							m.historyModel.LoadMessages(conv.ID)
							m.updateMessagesViewport()
						}
					}
				}
			}
			return m, nil
		case "esc":
			// Return to agents view
			if m.viewMode == "messages" || m.viewMode == "sync" || m.viewMode == "help" || m.viewMode == "ssh_connections" {
				m.viewMode = "agents"
			}
			return m, nil

		case "x":
			// Toggle to SSH connections view
			if m.viewMode == "agents" {
				m.viewMode = "ssh_connections"
				m.sshSelectedIndex = 0
				m.sshDeleteConfirm = false
				m.sshDeleteTarget = 0
			} else if m.viewMode == "ssh_connections" {
				m.viewMode = "agents"
			}
			return m, nil


		// Sync view navigation
		case "tab":
			if m.viewMode == "sync" {
				if m.syncMode == views.EditMode {
					// Exit edit mode
					m.syncMode = views.ViewMode
					m.syncEditor.Blur()
				} else {
					// Enter edit mode
					m.syncMode = views.EditMode
					m.syncEditor.Focus()
				}
			}
			return m, nil
		// 's' key removed - use 'e' for sync customization only
		case "c":
			if m.viewMode == "sync" && m.syncMode != views.EditMode {
				// Start file picker for custom sync
				m.filePickerMode = true
				m.filePickerLoading = true
				m.filePickerError = ""
				m.discoveredFiles = nil
				m.filePickerIndex = 0

				// Initialize 7 different spinners for file discovery
				spinnerTypes := []spinner.Spinner{
					spinner.Dot,
					spinner.Line,
					spinner.MiniDot,
					spinner.Jump,
					spinner.Pulse,
					spinner.Points,
					spinner.Globe,
				}

				colors := []string{"62", "196", "214", "34", "99", "208", "165"}

				m.filePickerSpinners = make([]spinner.Model, 7)
				var spinnerCmds []tea.Cmd

				for i := 0; i < 7; i++ {
					s := spinner.New()
					s.Spinner = spinnerTypes[i]
					s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(colors[i]))
					m.filePickerSpinners[i] = s
					spinnerCmds = append(spinnerCmds, m.filePickerSpinners[i].Tick)
				}

				// Add the file discovery command
				spinnerCmds = append(spinnerCmds, m.discoverFilesCommand())

				return m, tea.Batch(spinnerCmds...)
			}
			return m, nil
		case "left":
			if m.viewMode == "help" && m.helpModel != nil {
				// Switch to previous help tab
				m.helpModel.PrevTab()
				return m, nil
			} else if m.viewMode == "messages" {
				// Move focus to conversations panel when in messages view
				m.messagesFocus = "conversations"
				// Reset message selection (-1 means no selection)
				m.selectedMessage = -1
				if m.historyModel != nil {
					m.updateMessagesViewport()
				}
			}
			return m, nil
		case "right":
			if m.viewMode == "help" && m.helpModel != nil {
				// Switch to next help tab
				m.helpModel.NextTab()
				return m, nil
			} else if m.viewMode == "messages" {
				// Move focus to messages panel when in messages view
				m.messagesFocus = "messages"
				// Load messages with first one selected/untruncated
				if m.historyModel != nil {
					messages := m.historyModel.GetMessages()
					if len(messages) > 0 {
						m.selectedMessage = 0
						m.updateMessagesViewport()
						m.messagesViewport.GotoTop()
					}
				}
			}
			return m, nil
		case "up", "k":
			if m.viewMode == "help" && m.helpModel != nil {
				// Pass navigation to help viewport
				m.helpModel.UpdateViewport(msg)
				return m, nil
			} else if m.viewMode == "ssh_connections" {
				// Navigate SSH connections list
				if m.sshRegistry != nil && !m.sshDeleteConfirm {
					connCount := len(m.sshRegistry.GetConnections())
					if connCount > 0 && m.sshSelectedIndex > 0 {
						m.sshSelectedIndex--
					}
				}
				return m, nil
			} else if m.viewMode == "messages" {
				if m.messagesFocus == "conversations" {
					// Navigate conversations in left panel
					if m.historyModel != nil && m.historyModel.HasConversations() {
						if m.historyModel.SelectedConv > 0 {
							m.historyModel.SelectedConv--
							// Load messages for selected conversation
							conv := m.historyModel.GetSelectedConversation()
							if conv != nil {
								m.selectedMessage = -1  // Reset selection when changing conversations (-1 = no selection)
								m.historyModel.LoadMessages(conv.ID)
								m.updateMessagesViewport()
								m.messagesViewport.GotoTop()
							}
						}
					}
				} else if m.messagesFocus == "messages" {
					// Navigate individual messages in the list
					if m.historyModel != nil {
						messageCount := len(m.historyModel.GetMessages())
						if messageCount > 0 && m.selectedMessage > 0 {
							m.selectedMessage--
							m.updateMessagesViewport()
						}
					}
				}
				return m, nil
			} else if m.viewMode == "agents" {
				// Forward navigation to bubble-table
				var tableCmd tea.Cmd
				m.table, tableCmd = m.table.Update(msg)
				return m, tableCmd
			}
		case "down", "j":
			if m.viewMode == "help" && m.helpModel != nil {
				// Pass navigation to help viewport
				m.helpModel.UpdateViewport(msg)
				return m, nil
			} else if m.viewMode == "ssh_connections" {
				// Navigate SSH connections list
				if m.sshRegistry != nil && !m.sshDeleteConfirm {
					connCount := len(m.sshRegistry.GetConnections())
					if connCount > 0 && m.sshSelectedIndex < connCount-1 {
						m.sshSelectedIndex++
					}
				}
				return m, nil
			} else if m.viewMode == "messages" {
				if m.messagesFocus == "conversations" {
					// Navigate conversations in left panel
					if m.historyModel != nil && m.historyModel.HasConversations() {
						if m.historyModel.SelectedConv < m.historyModel.ConversationCount()-1 {
							m.historyModel.SelectedConv++
							// Load messages for selected conversation
							conv := m.historyModel.GetSelectedConversation()
							if conv != nil {
								m.selectedMessage = -1  // Reset selection when changing conversations (-1 = no selection)
								m.historyModel.LoadMessages(conv.ID)
								m.updateMessagesViewport()
								m.messagesViewport.GotoTop()
							}
						}
					}
				} else if m.messagesFocus == "messages" {
					// Navigate individual messages in the list
					if m.historyModel != nil {
						messageCount := len(m.historyModel.GetMessages())
						if messageCount > 0 && m.selectedMessage < messageCount-1 {
							m.selectedMessage++
							m.updateMessagesViewport()
						}
					}
				}
				return m, nil
			} else if m.viewMode == "agents" {
				// Forward navigation to bubble-table
				var tableCmd tea.Cmd
				m.table, tableCmd = m.table.Update(msg)
				return m, tableCmd
			}
		case "r":
			if m.viewMode == "agents" {
				// Manual refresh - sync everything
				m = m.refreshAll()
			} else if m.viewMode == "messages" {
				// Refresh message history
				if m.historyModel != nil {
					m.historyModel.LoadConversations()
					// Reload messages for current conversation if any
					if m.historyModel.HasConversations() && m.historyModel.SelectedConv < m.historyModel.ConversationCount() {
						conv := m.historyModel.GetSelectedConversation()
						if conv != nil {
							m.historyModel.LoadMessages(conv.ID)
							m.updateMessagesViewport()
						}
					}
				}
			}
			return m, nil
		case "e":
			if m.viewMode == "agents" {
				// Edit/customize sync clauses
				m.viewMode = "sync"
				m = m.initializeSyncComponents()
			}
		case "?":
			if m.viewMode == "agents" {
				// Show help view
				m.viewMode = "help"
				// Initialize help model if not already done
				if m.helpModel == nil {
					var err error
					m.helpModel, err = views.NewHelpModel(m.width, m.height)
					if err != nil {
						// Handle error gracefully - return to agents view
						m.viewMode = "agents"
						return m, nil
					}
				} else {
					// Update dimensions in case terminal was resized
					m.helpModel.Update(m.width, m.height)
				}
			}
		case "d":
			// Delete SSH connection when in ssh_connections view
			if m.viewMode == "ssh_connections" && !m.sshDeleteConfirm {
				if m.sshRegistry != nil {
					connCount := len(m.sshRegistry.GetConnections())
					if connCount > 0 && m.sshSelectedIndex < connCount {
						m.sshDeleteConfirm = true
						m.sshDeleteTarget = m.sshSelectedIndex
					}
				}
				return m, nil
			}
			// Delete conversation when in messages view and conversations panel has focus
			if m.viewMode == "messages" && m.messagesFocus == "conversations" && !m.deleteConfirm {
				if m.historyModel != nil && m.historyModel.HasConversations() {
					conv := m.historyModel.GetSelectedConversation()
					if conv != nil {
						m.deleteConfirm = true
						m.deleteTarget = conv.ID
					}
				}
			}
		case "y":
			// Confirm SSH connection deletion
			if m.sshDeleteConfirm {
				if m.sshRegistry != nil {
					connections := m.sshRegistry.GetConnections()
					if m.sshDeleteTarget < len(connections) {
						// Remove the connection
						targetName := connections[m.sshDeleteTarget].Name
						err := m.sshRegistry.RemoveConnection(targetName)
						if err == nil {
							// Adjust selection if needed
							connCount := len(m.sshRegistry.GetConnections())
							if m.sshSelectedIndex >= connCount && connCount > 0 {
								m.sshSelectedIndex = connCount - 1
							}
							// Refresh agents table to remove stale remote agents
							m = m.refreshAll()
						}
					}
				}
				m.sshDeleteConfirm = false
				m.sshDeleteTarget = 0
				return m, nil
			}
			// Confirm deletion
			if m.deleteConfirm {
				if m.historyModel != nil {
					err := m.historyModel.DeleteConversation(m.deleteTarget)
					if err == nil {
						// Successfully deleted, reload conversations
						m.historyModel.LoadConversations()
						// Clear message panel
						m.messagesViewport.SetContent("")
							}
				}
				m.deleteConfirm = false
				m.deleteTarget = 0
			}
		case "n":
			// Cancel SSH connection deletion
			if m.sshDeleteConfirm {
				m.sshDeleteConfirm = false
				m.sshDeleteTarget = 0
				return m, nil
			}
			// Cancel deletion
			if m.deleteConfirm {
				m.deleteConfirm = false
				m.deleteTarget = 0
			}
		case "a":
			// Register agent - enter input mode (only for local agents)
			selectedRowIndex := m.table.GetHighlightedRowIndex()
			if selectedRowIndex >= 0 && selectedRowIndex < len(m.rows) && len(m.rows) > 0 {
				row := m.rows[selectedRowIndex]
				if len(row) >= 7 {  // Make sure we have machine column
					agentType := row[2]     // AGENT column
					fullDirectory := row[1] // DIRECTORY column (full path)
					machine := row[5]       // MACHINE column

					// Only allow registration/deregistration for local agents
					if machine == "host" {
						if m.registry.IsRegisteredWithMachine(agentType, fullDirectory, machine) {
							// Already registered, deregister it
							m.registry.DeregisterWithMachine(agentType, fullDirectory, machine)
							// Refresh everything
							m = m.refreshAll()
						} else {
							// Enter input mode to get name
							m.inputMode = true
							m.inputBuffer = ""
							m.inputTarget = "register"
						}
					}
					// Ignore 'a' key for remote agents (machine != "host")
				}
			}
		case "z":
			// Register SSH connection - start multi-step input process (agents view only)
			if m.viewMode == "agents" && m.sshRegistry != nil {
				// Start SSH connection registration process
				m.inputMode = true
				m.inputBuffer = ""
				m.inputTarget = "ssh-name"
				// Clear temp SSH fields
				m.tempSSHName = ""
				m.tempSSHKey = ""
				m.tempSSHCommand = ""
			}
		case "pgup":
			if m.viewMode == "messages" && m.messagesFocus == "messages" {
				// Page up in messages viewport (scroll within current message)
				m.messagesViewport, cmd = m.messagesViewport.Update(msg)
				return m, cmd
			}
		case "pgdn":
			if m.viewMode == "messages" && m.messagesFocus == "messages" {
				// Page down in messages viewport (scroll within current message)
				m.messagesViewport, cmd = m.messagesViewport.Update(msg)
				return m, cmd
			}
		}
	}

	// Handle textarea updates when in sync edit mode
	if m.viewMode == "sync" && m.syncMode == views.EditMode {
		oldValue := m.syncEditor.Value()
		updatedEditor, editorCmd := m.syncEditor.Update(msg)
		m.syncEditor = updatedEditor
		cmd = tea.Batch(cmd, editorCmd)
		// Mark as modified if content changed
		if oldValue != updatedEditor.Value() {
			m.syncModified = true
		}
	}

	return m, cmd
}