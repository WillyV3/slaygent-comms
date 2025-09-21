package views

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"slaygent-manager/history"
)

var (
	// Messages view styling constants
	messagesTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87CEEB")).
		Bold(true)

	messagesControlsStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	focusedBorderColor   = lipgloss.Color("#87CEEB")
	unfocusedBorderColor = lipgloss.Color("#006666")

	panelStyle = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder())

	confirmDialogStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF6B6B")).
		Background(lipgloss.Color("#2A2A2A")).
		Foreground(lipgloss.Color("#87CEEB")).
		Padding(1, 2).
		Bold(true)
)

// MessagesViewData contains all data needed to render the messages view
type MessagesViewData struct {
	HistoryModel     *history.Model
	MessagesViewport viewport.Model
	MessagesFocus    string // "conversations" or "messages"
	SelectedMessage  int
	DeleteConfirm    bool   // Whether delete confirmation is active
	DeleteTarget     int    // ID of conversation to delete
	Width            int
	Height           int
}

// RenderMessagesView renders the messages view
func RenderMessagesView(data MessagesViewData) string {
	if data.HistoryModel == nil {
		return "\nDatabase unavailable\n\nPress ESC to return\n"
	}

	// Simple calculations - do once at top
	leftWidth := data.Width / 3
	if leftWidth < 25 { leftWidth = 25 }
	rightWidth := data.Width - leftWidth - 6
	panelHeight := data.Height - 8

	// Simple header
	title := messagesTitleStyle.Render("MESSAGE HISTORY")

	// Simple controls
	controls := messagesControlsStyle.Render("↑/↓: navigate • ←/→: switch panels • d: delete • ESC: back")

	// Build panels
	leftPanel := renderConversationsPanel(data, leftWidth, panelHeight)
	rightPanel := renderMessagesPanel(data, rightWidth, panelHeight)

	// Assemble view
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)
	view := "\n" + title + "\n\n" + content + "\n\n" + controls

	// Handle delete confirmation overlay
	if data.DeleteConfirm {
		view = renderDeleteConfirmation(data)
	}

	return wrapToTerminal(view, data.Width)
}

// Simple helper functions
func renderConversationsPanel(data MessagesViewData, width, height int) string {
	content := data.HistoryModel.FormatConversationListWithSelection()
	borderColor := unfocusedBorderColor
	if data.MessagesFocus == "conversations" {
		borderColor = focusedBorderColor
	}

	return panelStyle.
		Width(width).
		Height(height).
		BorderForeground(borderColor).
		Render(content)
}

func renderMessagesPanel(data MessagesViewData, width, height int) string {
	content := data.MessagesViewport.View()
	borderColor := unfocusedBorderColor
	if data.MessagesFocus == "messages" {
		borderColor = focusedBorderColor
	}

	return panelStyle.
		Width(width).
		Height(height).
		BorderForeground(borderColor).
		Render(content)
}

func renderDeleteConfirmation(data MessagesViewData) string {
	var message string
	if conv := data.HistoryModel.GetSelectedConversation(); conv != nil {
		message = fmt.Sprintf("Delete conversation between %s and %s?", conv.Agent1Name, conv.Agent2Name)
	} else {
		message = "Delete selected conversation?"
	}

	confirmText := fmt.Sprintf("%s\n\nPress 'y' to confirm, 'n' to cancel", message)
	dialog := confirmDialogStyle.Render(confirmText)

	return lipgloss.Place(data.Width, data.Height, lipgloss.Center, lipgloss.Center, dialog)
}

