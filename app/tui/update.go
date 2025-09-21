package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/progress"
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
	case progress.FrameMsg:
		if m.syncing {
			progressModel, cmd := m.progress.Update(msg)
			m.progress = progressModel.(progress.Model)
			return m, cmd
		}
		return m, nil
	case refreshMsg:
		// Auto-refresh disabled to prevent duplication
		// Use manual refresh with 'r' key only
	case tea.KeyMsg:
		// Handle sync confirmation mode
		if m.syncConfirm {
			switch msg.String() {
			case "y", "Y":
				// Start sync with progress animation
				m.syncConfirm = false
				m.syncing = true
				m.progress.SetPercent(0)
				return m, tea.Batch(syncTickCmd(), m.runSyncCommand())
			case "n", "N", "esc":
				m.syncConfirm = false
				return m, nil
			}
			return m, nil
		}

		// Handle input mode first
		if m.inputMode {
			switch msg.String() {
			case "enter":
				// Save the registration with the entered name
				selectedRowIndex := m.table.GetHighlightedRowIndex()
				if m.inputBuffer != "" && selectedRowIndex >= 0 && selectedRowIndex < len(m.rows) {
					row := m.rows[selectedRowIndex]
					if len(row) >= 3 {
						agentType := row[2]  // AGENT column
						fullDirectory := row[1]  // DIRECTORY column (full path)
						m.registry.Register(m.inputBuffer, agentType, fullDirectory)
					}
				}
				// Exit input mode
				m.inputMode = false
				m.inputBuffer = ""
				m.inputTarget = ""
				// Refresh everything
				m = m.refreshAll()
			case "esc":
				// Cancel input mode
				m.inputMode = false
				m.inputBuffer = ""
				m.inputTarget = ""
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
			if m.viewMode == "messages" || m.viewMode == "sync" || m.viewMode == "help" {
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
		case "s":
			if m.viewMode == "agents" {
				// Quick sync CLAUDE.md files with registry
				m.syncConfirm = true
			}
			return m, nil
		case "c":
			if m.viewMode == "sync" && m.syncMode != views.EditMode {
				// Custom sync - switch to agents view and show progress there
				m.syncModified = false
				m.viewMode = "agents"  // Switch back to agents view
				m.syncing = true
				m.progress.SetPercent(0)
				return m, tea.Batch(syncTickCmd(), m.runCustomSyncCommand())
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
			// Cancel deletion
			if m.deleteConfirm {
				m.deleteConfirm = false
				m.deleteTarget = 0
			}
		case "a":
			// Register agent - enter input mode
			selectedRowIndex := m.table.GetHighlightedRowIndex()
			if selectedRowIndex >= 0 && selectedRowIndex < len(m.rows) && len(m.rows) > 0 {
				row := m.rows[selectedRowIndex]
				if len(row) >= 3 {
					agentType := row[2]
					fullDirectory := row[1]  // Full path for registry

					if m.registry.IsRegistered(agentType, fullDirectory) {
						// Already registered, deregister it
						m.registry.Deregister(agentType, fullDirectory)
						// Refresh everything
						m = m.refreshAll()
					} else {
						// Enter input mode to get name
						m.inputMode = true
						m.inputBuffer = ""
						m.inputTarget = "register"
					}
				}
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