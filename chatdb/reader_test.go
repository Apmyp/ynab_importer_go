package chatdb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestNewReader_NonExistentDB(t *testing.T) {
	_, err := NewReader("/nonexistent/path/chat.db", []string{"102"})
	if err == nil {
		t.Error("NewReader() should return error for non-existent database")
	}
}

func TestNewReader_ValidDB(t *testing.T) {
	dbPath := createTestDB(t)
	defer os.Remove(dbPath)

	reader, err := NewReader(dbPath, []string{"102"})
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer reader.Close()

	if reader == nil {
		t.Error("NewReader() returned nil reader")
	}
}

func TestReader_Close(t *testing.T) {
	dbPath := createTestDB(t)
	defer os.Remove(dbPath)

	reader, err := NewReader(dbPath, []string{"102"})
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}

	err = reader.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestReader_FetchMessages_EmptyDB(t *testing.T) {
	dbPath := createTestDB(t)
	defer os.Remove(dbPath)

	reader, err := NewReader(dbPath, []string{"102"})
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer reader.Close()

	messages, err := reader.FetchMessages()
	if err != nil {
		t.Fatalf("FetchMessages() error = %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("expected 0 messages from empty database, got %d", len(messages))
	}
}

func TestReader_FetchMessages_WithData(t *testing.T) {
	dbPath := createTestDBWithData(t)
	defer os.Remove(dbPath)

	reader, err := NewReader(dbPath, []string{"102"})
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer reader.Close()

	messages, err := reader.FetchMessages()
	if err != nil {
		t.Fatalf("FetchMessages() error = %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	// Verify first message
	if messages[0].Sender != "102" {
		t.Errorf("expected sender '102', got %q", messages[0].Sender)
	}
	if messages[0].Content != "Test message 1" {
		t.Errorf("expected content 'Test message 1', got %q", messages[0].Content)
	}

	// Verify second message
	if messages[1].Sender != "102" {
		t.Errorf("expected sender '102', got %q", messages[1].Sender)
	}
	if messages[1].Content != "Test message 2" {
		t.Errorf("expected content 'Test message 2', got %q", messages[1].Content)
	}
}

func TestReader_FetchMessages_MultipleSenders(t *testing.T) {
	dbPath := createTestDBWithMultipleSenders(t)
	defer os.Remove(dbPath)

	reader, err := NewReader(dbPath, []string{"102", "EXIMBANK"})
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer reader.Close()

	messages, err := reader.FetchMessages()
	if err != nil {
		t.Fatalf("FetchMessages() error = %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	// Count messages by sender
	senderCounts := make(map[string]int)
	for _, msg := range messages {
		senderCounts[msg.Sender]++
	}

	if senderCounts["102"] != 2 {
		t.Errorf("expected 2 messages from 102, got %d", senderCounts["102"])
	}
	if senderCounts["EXIMBANK"] != 1 {
		t.Errorf("expected 1 message from EXIMBANK, got %d", senderCounts["EXIMBANK"])
	}
}

func TestReader_FetchMessages_ExcludesOwnMessages(t *testing.T) {
	dbPath := createTestDBWithOwnMessages(t)
	defer os.Remove(dbPath)

	reader, err := NewReader(dbPath, []string{"102"})
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer reader.Close()

	messages, err := reader.FetchMessages()
	if err != nil {
		t.Fatalf("FetchMessages() error = %v", err)
	}

	// Should only include received messages, not sent ones
	for _, msg := range messages {
		if msg.Sender == "Me" {
			t.Error("FetchMessages() should not include own messages")
		}
	}
}

func TestAppleTimeToUnix(t *testing.T) {
	testCases := []struct {
		name      string
		appleTime int64
		expected  time.Time
	}{
		{
			name:      "zero time (2001-01-01)",
			appleTime: 0,
			expected:  time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "positive time in nanoseconds",
			appleTime: 704823707000000000, // 2023-05-03 16:21:47
			expected:  time.Date(2023, 5, 3, 16, 21, 47, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := appleTimeToUnix(tc.appleTime)
			if !result.Equal(tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// Helper functions to create test databases

func createTestDB(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_chat.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	// Create minimal schema matching Messages database
	schema := `
		CREATE TABLE handle (
			ROWID INTEGER PRIMARY KEY,
			id TEXT
		);

		CREATE TABLE message (
			ROWID INTEGER PRIMARY KEY,
			handle_id INTEGER,
			text TEXT,
			date INTEGER,
			is_from_me INTEGER
		);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return dbPath
}

func createTestDBWithData(t *testing.T) string {
	t.Helper()

	dbPath := createTestDB(t)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	// Insert test handle
	_, err = db.Exec("INSERT INTO handle (ROWID, id) VALUES (1, '102')")
	if err != nil {
		t.Fatalf("failed to insert handle: %v", err)
	}

	// Insert test messages
	// Apple time: nanoseconds since 2001-01-01
	// 704823707000000000 = 2023-05-03 16:21:47
	_, err = db.Exec(`
		INSERT INTO message (ROWID, handle_id, text, date, is_from_me)
		VALUES
			(1, 1, 'Test message 1', 704823707000000000, 0),
			(2, 1, 'Test message 2', 704823708000000000, 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert messages: %v", err)
	}

	return dbPath
}

func createTestDBWithMultipleSenders(t *testing.T) string {
	t.Helper()

	dbPath := createTestDB(t)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	// Insert test handles
	_, err = db.Exec(`
		INSERT INTO handle (ROWID, id) VALUES
			(1, '102'),
			(2, 'EXIMBANK')
	`)
	if err != nil {
		t.Fatalf("failed to insert handles: %v", err)
	}

	// Insert test messages
	_, err = db.Exec(`
		INSERT INTO message (ROWID, handle_id, text, date, is_from_me)
		VALUES
			(1, 1, 'Message from 102', 704823707000000000, 0),
			(2, 2, 'Message from EXIMBANK', 704823708000000000, 0),
			(3, 1, 'Another from 102', 704823709000000000, 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert messages: %v", err)
	}

	return dbPath
}

func createTestDBWithOwnMessages(t *testing.T) string {
	t.Helper()

	dbPath := createTestDB(t)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	// Insert test handle
	_, err = db.Exec("INSERT INTO handle (ROWID, id) VALUES (1, '102')")
	if err != nil {
		t.Fatalf("failed to insert handle: %v", err)
	}

	// Insert test messages including own messages
	_, err = db.Exec(`
		INSERT INTO message (ROWID, handle_id, text, date, is_from_me)
		VALUES
			(1, 1, 'Received message', 704823707000000000, 0),
			(2, 1, 'Sent message', 704823708000000000, 1),
			(3, 1, 'Another received', 704823709000000000, 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert messages: %v", err)
	}

	return dbPath
}

func TestReader_FetchMessages_EmptyTextMessages(t *testing.T) {
	dbPath := createTestDB(t)
	defer os.Remove(dbPath)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	// Insert handle and messages with null/empty text
	_, err = db.Exec("INSERT INTO handle (ROWID, id) VALUES (1, '102')")
	if err != nil {
		t.Fatalf("failed to insert handle: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO message (ROWID, handle_id, text, date, is_from_me)
		VALUES
			(1, 1, NULL, 704823707000000000, 0),
			(2, 1, '', 704823708000000000, 0),
			(3, 1, 'Valid message', 704823709000000000, 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert messages: %v", err)
	}

	reader, err := NewReader(dbPath, []string{"102"})
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer reader.Close()

	messages, err := reader.FetchMessages()
	if err != nil {
		t.Fatalf("FetchMessages() error = %v", err)
	}

	// Should only get the valid message
	if len(messages) != 1 {
		t.Errorf("expected 1 message (skipping null/empty), got %d", len(messages))
	}
	if len(messages) > 0 && messages[0].Content != "Valid message" {
		t.Errorf("expected 'Valid message', got %q", messages[0].Content)
	}
}

func TestReader_FetchMessages_NoSenders(t *testing.T) {
	dbPath := createTestDB(t)
	defer os.Remove(dbPath)

	reader, err := NewReader(dbPath, []string{})
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer reader.Close()

	messages, err := reader.FetchMessages()
	if err != nil {
		t.Fatalf("FetchMessages() error = %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("expected 0 messages with no senders, got %d", len(messages))
	}
}

func TestReader_Close_NilDB(t *testing.T) {
	reader := &Reader{db: nil, senders: []string{}}
	err := reader.Close()
	if err != nil {
		t.Errorf("Close() should not error with nil db: %v", err)
	}
}

func TestBuildPlaceholders_Zero(t *testing.T) {
	result := buildPlaceholders(0)
	if result != "" {
		t.Errorf("buildPlaceholders(0) expected empty string, got %q", result)
	}
}

func TestBuildPlaceholders_Multiple(t *testing.T) {
	testCases := []struct {
		count    int
		expected string
	}{
		{1, "?"},
		{2, "?,?"},
		{3, "?,?,?"},
		{5, "?,?,?,?,?"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("count=%d", tc.count), func(t *testing.T) {
			result := buildPlaceholders(tc.count)
			if result != tc.expected {
				t.Errorf("buildPlaceholders(%d) = %q, want %q", tc.count, result, tc.expected)
			}
		})
	}
}
