package history

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	_ "github.com/mattn/go-sqlite3"
)

type Conversation struct {
	ID           int
	Agent1Name   string
	Agent1Dir    string
	Agent2Name   string
	Agent2Dir    string
	LastMessage  time.Time
	MessageCount int
}

type Message struct {
	SenderName   string
	SenderDir    string
	ReceiverName string
	ReceiverDir  string
	Message      string
	SentAt       time.Time
}

type SyncClause struct {
	ID          int
	ClauseType  string
	Content     string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Model struct {
	db            *sql.DB
	conversations []Conversation
	messages      []Message
	SelectedConv  int
}

func New(dbPath string) (*Model, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return &Model{db: db}, nil
}

func (m *Model) LoadConversations() error {
	query := `
		SELECT c.id, c.agent1_name, c.agent1_dir, c.agent2_name, c.agent2_dir,
		       c.last_message_at,
		       (SELECT COUNT(*) FROM messages WHERE conversation_id = c.id) as msg_count
		FROM conversations c
		ORDER BY c.last_message_at DESC
		LIMIT 100`

	rows, err := m.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	m.conversations = nil
	for rows.Next() {
		var conv Conversation
		err := rows.Scan(&conv.ID, &conv.Agent1Name, &conv.Agent1Dir,
			&conv.Agent2Name, &conv.Agent2Dir, &conv.LastMessage, &conv.MessageCount)
		if err != nil {
			return err
		}
		m.conversations = append(m.conversations, conv)
	}

	return rows.Err()
}

func (m *Model) LoadMessages(conversationID int) error {
	query := `
		SELECT sender_name, sender_dir, receiver_name, receiver_dir,
		       message, sent_at
		FROM messages
		WHERE conversation_id = ?
		ORDER BY sent_at ASC`

	rows, err := m.db.Query(query, conversationID)
	if err != nil {
		return err
	}
	defer rows.Close()

	m.messages = nil
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.SenderName, &msg.SenderDir,
			&msg.ReceiverName, &msg.ReceiverDir, &msg.Message, &msg.SentAt)
		if err != nil {
			return err
		}
		m.messages = append(m.messages, msg)
	}

	return rows.Err()
}

func (m *Model) Close() {
	if m.db != nil {
		m.db.Close()
	}
}

func (m *Model) DeleteConversation(conversationID int) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Start transaction for atomic deletion
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Will be ignored if transaction is committed

	// First delete all messages in the conversation (foreign key constraint)
	_, err = tx.Exec("DELETE FROM messages WHERE conversation_id = ?", conversationID)
	if err != nil {
		return err
	}

	// Then delete the conversation itself
	_, err = tx.Exec("DELETE FROM conversations WHERE id = ?", conversationID)
	if err != nil {
		return err
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return err
	}

	// Remove from local conversations slice
	for i, conv := range m.conversations {
		if conv.ID == conversationID {
			m.conversations = append(m.conversations[:i], m.conversations[i+1:]...)
			// Adjust selection if needed
			if m.SelectedConv >= len(m.conversations) && len(m.conversations) > 0 {
				m.SelectedConv = len(m.conversations) - 1
			} else if len(m.conversations) == 0 {
				m.SelectedConv = 0
			}
			break
		}
	}

	// Clear messages if this was the selected conversation
	m.messages = nil

	return nil
}

func (m *Model) FormatConversationList() string {
	if len(m.conversations) == 0 {
		return "No conversations found"
	}

	var lines []string
	for i, conv := range m.conversations {
		prefix := "  "
		if i == m.SelectedConv {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%s ↔ %s | %s | %d msgs",
			prefix,
			conv.Agent1Name,
			conv.Agent2Name,
			conv.LastMessage.Format("Jan 02 15:04"),
			conv.MessageCount)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m *Model) FormatConversationListWithSelection() string {
	if len(m.conversations) == 0 {
		return "No conversations found"
	}

	var lines []string
	var lastTimeTag string
	now := time.Now()

	for i, conv := range m.conversations {
		// Calculate relative time tag
		timeTag := getRelativeTimeTag(conv.LastMessage, now)

		// Only show time tag if it's different from the last one
		if timeTag != lastTimeTag {
			// Add the time tag as a header
			styledTag := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666")).
				Bold(true).
				Render(fmt.Sprintf("[%s]", timeTag))
			lines = append(lines, styledTag)
			lastTimeTag = timeTag
		}

		prefix := "  "
		if i == m.SelectedConv {
			prefix = "> "
		}

		// First agent gets baby blue, second gets green
		styledAgent1 := lipgloss.NewStyle().Foreground(lipgloss.Color("#87CEEB")).Render(conv.Agent1Name)
		styledAgent2 := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render(conv.Agent2Name)

		line := fmt.Sprintf("%s%s ↔ %s",
			prefix,
			styledAgent1,
			styledAgent2)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func getRelativeTimeTag(t time.Time, now time.Time) string {
	// Use date-based comparison for more accurate day boundaries
	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tDate := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	daysDiff := int(nowDate.Sub(tDate).Hours() / 24)

	if daysDiff == 0 {
		return "today"
	} else if daysDiff == 1 {
		return "yesterday"
	} else if daysDiff <= 7 {
		return fmt.Sprintf("%d days ago", daysDiff)
	} else if daysDiff <= 14 {
		return "last week"
	} else if daysDiff <= 30 {
		weeks := daysDiff / 7
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if daysDiff <= 365 {
		months := daysDiff / 30
		if months == 1 {
			return "last month"
		}
		return fmt.Sprintf("%d months ago", months)
	}
	return "over a year ago"
}

func (m *Model) FormatMessages() string {
	if len(m.messages) == 0 {
		return "No messages in this conversation"
	}

	// Get the first agent in this conversation (for consistent coloring)
	var agent1 string
	if len(m.messages) > 0 {
		agent1 = m.messages[0].SenderName
	}

	var lines []string
	for _, msg := range m.messages {
		timestamp := msg.SentAt.Format("15:04:05")
		styledTimestamp := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Faint(true).Render(fmt.Sprintf("[%s]", timestamp))

		// Agent1 gets baby blue, Agent2 gets green
		senderColor := lipgloss.Color("#00FF00") // Default green
		if msg.SenderName == agent1 {
			senderColor = lipgloss.Color("#87CEEB") // Baby blue
		}

		styledSender := lipgloss.NewStyle().Foreground(senderColor).Render(msg.SenderName)
		styledReceiver := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(msg.ReceiverName)
		styledMessage := lipgloss.NewStyle().Foreground(senderColor).Faint(true).Render(msg.Message)

		line := fmt.Sprintf("%s %s → %s: %s",
			styledTimestamp,
			styledSender,
			styledReceiver,
			styledMessage)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m *Model) GetSelectedConversation() *Conversation {
	if m.SelectedConv >= 0 && m.SelectedConv < len(m.conversations) {
		return &m.conversations[m.SelectedConv]
	}
	return nil
}

func (m *Model) HasConversations() bool {
	return len(m.conversations) > 0
}

func (m *Model) ConversationCount() int {
	return len(m.conversations)
}

func (m *Model) GetMessages() []Message {
	return m.messages
}

func (m *Model) GetConversations() []Conversation {
	return m.conversations
}

func (m *Model) FormatMessagesWithSelection(selectedMessage int) string {
	if len(m.messages) == 0 {
		return "No messages in this conversation"
	}

	// Get the first agent in this conversation (for consistent coloring)
	var agent1 string
	if len(m.messages) > 0 {
		agent1 = m.messages[0].SenderName
	}

	var lines []string
	for i, msg := range m.messages {
		timestamp := msg.SentAt.Format("15:04:05")
		styledTimestamp := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Faint(true).Render(fmt.Sprintf("[%s]", timestamp))

		// Agent1 gets baby blue, Agent2 gets green
		senderColor := lipgloss.Color("#00FF00") // Default green
		if msg.SenderName == agent1 {
			senderColor = lipgloss.Color("#87CEEB") // Baby blue
		}

		styledSender := lipgloss.NewStyle().Foreground(senderColor).Render(msg.SenderName)
		styledReceiver := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(msg.ReceiverName)

		// Show full message for selected, normal for others
		var styledMessage string
		if i == selectedMessage {
			// Full message, bold and highlighted
			styledMessage = lipgloss.NewStyle().Foreground(senderColor).Bold(true).Render(msg.Message)
		} else {
			styledMessage = lipgloss.NewStyle().Foreground(senderColor).Faint(true).Render(msg.Message)
		}

		line := fmt.Sprintf("%s %s → %s: %s",
			styledTimestamp,
			styledSender,
			styledReceiver,
			styledMessage)

		// Highlight selected message background
		if i == selectedMessage {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("#444444")).
				Render(line)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}


