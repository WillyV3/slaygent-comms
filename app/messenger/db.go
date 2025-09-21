package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func InitDB() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	slaygentDir := filepath.Join(home, ".slaygent")
	os.MkdirAll(slaygentDir, 0755)

	dbPath := filepath.Join(slaygentDir, "messages.db")

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// Create tables if they don't exist
	schema := `
	CREATE TABLE IF NOT EXISTS conversations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent1_name TEXT NOT NULL,
		agent1_dir TEXT NOT NULL,
		agent2_name TEXT NOT NULL,
		agent2_dir TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_message_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(agent1_name, agent1_dir, agent2_name, agent2_dir)
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conversation_id INTEGER NOT NULL,
		sender_name TEXT NOT NULL,
		sender_dir TEXT NOT NULL,
		receiver_name TEXT NOT NULL,
		receiver_dir TEXT NOT NULL,
		message TEXT NOT NULL,
		sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (conversation_id) REFERENCES conversations(id)
	);

	CREATE INDEX IF NOT EXISTS idx_messages_sent_at ON messages(sent_at);
	`

	_, err = db.Exec(schema)
	if err != nil {
		return err
	}

	// Run cleanup on startup
	CleanupOldMessages()

	return nil
}

func getOrCreateConversation(sender *RegistryEntry, receiver *RegistryEntry) (int64, error) {
	// Sort agents alphabetically for consistent conversation grouping
	agents := []struct {
		Name string
		Dir  string
	}{
		{sender.Name, sender.Directory},
		{receiver.Name, receiver.Directory},
	}

	sort.Slice(agents, func(i, j int) bool {
		if agents[i].Name == agents[j].Name {
			return agents[i].Dir < agents[j].Dir
		}
		return agents[i].Name < agents[j].Name
	})

	// Check if conversation exists
	var conversationID int64
	err := db.QueryRow(`
		SELECT id FROM conversations
		WHERE agent1_name = ? AND agent1_dir = ?
		AND agent2_name = ? AND agent2_dir = ?`,
		agents[0].Name, agents[0].Dir,
		agents[1].Name, agents[1].Dir,
	).Scan(&conversationID)

	if err == sql.ErrNoRows {
		// Create new conversation
		result, err := db.Exec(`
			INSERT INTO conversations (agent1_name, agent1_dir, agent2_name, agent2_dir)
			VALUES (?, ?, ?, ?)`,
			agents[0].Name, agents[0].Dir,
			agents[1].Name, agents[1].Dir,
		)
		if err != nil {
			return 0, err
		}
		conversationID, err = result.LastInsertId()
		if err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}

	// Update last message timestamp
	_, err = db.Exec(`
		UPDATE conversations
		SET last_message_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		conversationID,
	)

	return conversationID, err
}

func LogMessage(sender, senderDir, receiver, receiverDir, message string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Create registry entries for conversation lookup
	senderEntry := &RegistryEntry{
		Name:      sender,
		Directory: senderDir,
	}
	receiverEntry := &RegistryEntry{
		Name:      receiver,
		Directory: receiverDir,
	}

	// Get or create conversation
	conversationID, err := getOrCreateConversation(senderEntry, receiverEntry)
	if err != nil {
		return err
	}

	// Insert message
	_, err = db.Exec(`
		INSERT INTO messages (conversation_id, sender_name, sender_dir, receiver_name, receiver_dir, message)
		VALUES (?, ?, ?, ?, ?, ?)`,
		conversationID, sender, senderDir, receiver, receiverDir, message,
	)

	return err
}

func LogMessageFromRegistry(senderInfo string, receiver *RegistryEntry, message string, registry []RegistryEntry) error {
	// Parse sender info
	var senderName, senderDir string

	// Find sender in registry
	for _, agent := range registry {
		if agent.Name == senderInfo {
			senderName = agent.Name
			senderDir = agent.Directory
			break
		}
	}

	// If not found in registry, don't log
	if senderName == "" {
		return nil // Silent failure for unknown senders
	}

	return LogMessage(senderName, senderDir, receiver.Name, receiver.Directory, message)
}

func LogMessageExplicit(senderName string, receiver *RegistryEntry, message string, registry []RegistryEntry) error {
	// Find sender in registry to get their directory
	var senderDir string
	for _, agent := range registry {
		if agent.Name == senderName {
			senderDir = agent.Directory
			break
		}
	}

	// If sender not in registry, use a placeholder directory
	if senderDir == "" {
		// Still log the message but with unknown directory
		senderDir = "unknown"
	}

	return LogMessage(senderName, senderDir, receiver.Name, receiver.Directory, message)
}

func ConversationExists(agent1Name, agent2Name string) bool {
	if db == nil {
		return false
	}

	// Sort names for consistent lookup
	names := []string{agent1Name, agent2Name}
	sort.Strings(names)

	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM conversations
		WHERE (agent1_name = ? AND agent2_name = ?)
		OR (agent1_name = ? AND agent2_name = ?)`,
		names[0], names[1], names[1], names[0],
	).Scan(&count)

	if err != nil {
		return false
	}

	return count > 0
}

func CleanupOldMessages() error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Delete messages older than 30 days
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Format("2006-01-02 15:04:05")

	result, err := db.Exec(`
		DELETE FROM messages
		WHERE sent_at < ?`,
		thirtyDaysAgo,
	)

	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Fprintf(os.Stderr, "Cleaned up %d old messages\n", rowsAffected)
	}

	return nil
}

func CloseDB() {
	if db != nil {
		db.Close()
	}
}

// Helper to get current sender's directory for logging
func getCurrentSenderDir() string {
	cmd := strings.Join(os.Args, " ")
	if strings.Contains(cmd, "tmux") {
		// We're being called from tmux context
		// This would need tmux API calls to get current pane directory
		return ""
	}

	// Default to current working directory
	dir, _ := os.Getwd()
	return dir
}