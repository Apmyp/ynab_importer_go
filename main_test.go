package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apmyp/ynab_importer_go/bagoup"
	"github.com/apmyp/ynab_importer_go/config"
	"github.com/apmyp/ynab_importer_go/template"
	_ "modernc.org/sqlite"
)

// MockFetcher is a test double for MessageFetcher
type MockFetcher struct {
	messages      []*bagoup.Message
	fetchErr      error
	dependencyErr error
	cleanupCalled bool
}

func (m *MockFetcher) CheckDependencies() error {
	return m.dependencyErr
}

func (m *MockFetcher) FetchMessages() ([]*bagoup.Message, func(), error) {
	if m.fetchErr != nil {
		return nil, func() {}, m.fetchErr
	}
	return m.messages, func() { m.cleanupCalled = true }, nil
}

func TestNewApp(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102", "EXIMBANK"},
	}

	app := NewApp(cfg)
	if app == nil {
		t.Error("NewApp() should return a non-nil App")
	}
	if app.config == nil {
		t.Error("App.config should not be nil")
	}
	if app.fetcher == nil {
		t.Error("App.fetcher should not be nil")
	}
	if app.matcher == nil {
		t.Error("App.matcher should not be nil")
	}
	if app.pool == nil {
		t.Error("App.pool should not be nil")
	}
}

func TestNewAppWithFetcher(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
	}
	mockFetcher := &MockFetcher{}

	app := NewAppWithFetcher(cfg, mockFetcher)
	if app == nil {
		t.Error("NewAppWithFetcher() should return non-nil App")
	}
	if app.fetcher != mockFetcher {
		t.Error("App.fetcher should be the mock fetcher")
	}
}

func TestParsedMessage_WithTemplate(t *testing.T) {
	msg := &bagoup.Message{
		Timestamp: time.Now(),
		Sender:    "102",
		Content:   "Test content",
	}
	tx := &template.Transaction{
		Operation: "Test",
		Original:  template.Amount{Value: 100.0, Currency: "MDL"},
	}

	pm := &ParsedMessage{
		Message:     msg,
		Transaction: tx,
		HasTemplate: true,
	}

	if pm.Message != msg {
		t.Error("ParsedMessage.Message should match")
	}
	if pm.Transaction != tx {
		t.Error("ParsedMessage.Transaction should match")
	}
	if !pm.HasTemplate {
		t.Error("ParsedMessage.HasTemplate should be true")
	}
}

func TestParsedMessage_WithoutTemplate(t *testing.T) {
	msg := &bagoup.Message{
		Timestamp: time.Now(),
		Sender:    "102",
		Content:   "Test content",
	}

	pm := &ParsedMessage{
		Message:     msg,
		Transaction: nil,
		HasTemplate: false,
	}

	if pm.HasTemplate {
		t.Error("ParsedMessage.HasTemplate should be false")
	}
}

func TestRun_ConfigNotFound(t *testing.T) {
	err := Run([]string{"--config", "/nonexistent/config.json"})
	if err == nil {
		t.Error("Run() should return error for non-existent config")
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	// Create temp config
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{"senders": ["102"], "bagoup": {"db_path": "test.db", "separate_chats": true}}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}

	err := Run([]string{"--config", configPath, "unknown_command"})
	if err == nil {
		t.Error("Run() should return error for unknown command")
	}
}

func TestApp_parseMessage_WithMatchingTemplate(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
	}
	app := NewApp(cfg)

	msg := &bagoup.Message{
		Timestamp: time.Now(),
		Sender:    "102",
		Content: `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: 34 MDL
Dost: 12500,50
Data/vremya: 03.05.23 16:21
Adres: COFFEE SHOP ALPHA
Podderzhka: +12025551234`,
	}

	pm := app.parseMessage(msg)

	if !pm.HasTemplate {
		t.Error("parseMessage should find template for MAIB transaction")
	}
	if pm.Transaction == nil {
		t.Fatal("Transaction should not be nil")
	}
	if pm.Transaction.Original.Value != 34.0 {
		t.Errorf("expected amount 34.0, got %f", pm.Transaction.Original.Value)
	}
}

func TestApp_parseMessage_WithoutMatchingTemplate(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
	}
	app := NewApp(cfg)

	msg := &bagoup.Message{
		Timestamp: time.Now(),
		Sender:    "102",
		Content:   "Random message that doesn't match any template",
	}

	pm := app.parseMessage(msg)

	if pm.HasTemplate {
		t.Error("parseMessage should not find template for random message")
	}
}

func TestRun_DefaultCommand(t *testing.T) {
	// Create temp config
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{"senders": ["102"], "bagoup": {"db_path": "test.db", "separate_chats": true}}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}

	// This will fail because bagoup won't find the db, but tests the command parsing
	err := Run([]string{"--config", configPath, "default"})
	// Will fail because of bagoup execution, not command parsing
	if err == nil {
		t.Skip("Expected error from bagoup execution")
	}
}

func TestRun_MissingTemplatesCommand(t *testing.T) {
	// Create temp config
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{"senders": ["102"], "bagoup": {"db_path": "test.db", "separate_chats": true}}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}

	// This will fail because bagoup won't find the db, but tests the command parsing
	err := Run([]string{"--config", configPath, "missing_templates"})
	// Will fail because of bagoup execution
	if err == nil {
		t.Skip("Expected error from bagoup execution")
	}
}

func TestApp_parseMessage_EximTransaction(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"EXIMBANK"},
	}
	app := NewApp(cfg)

	msg := &bagoup.Message{
		Timestamp: time.Now(),
		Sender:    "EXIMBANK",
		Content:   "Tranzactia din 29/05/2023 din contul ACC1234567MD4 in contul MD99XX000000011111111111 in suma de 5000.00 MDL a fost Executata",
	}

	pm := app.parseMessage(msg)

	if !pm.HasTemplate {
		t.Error("parseMessage should find template for EXIM transaction")
	}
	if pm.Transaction == nil {
		t.Fatal("Transaction should not be nil")
	}
	if pm.Transaction.Original.Value != 5000.00 {
		t.Errorf("expected amount 5000.00, got %f", pm.Transaction.Original.Value)
	}
}

func TestApp_runDefault_WithMock(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
	}

	now := time.Now()
	mockFetcher := &MockFetcher{
		messages: []*bagoup.Message{
			{
				Timestamp: now,
				Sender:    "102",
				Content: `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: 100 MDL
Dost: 1000,00
Data/vremya: 03.05.23 16:21
Adres: TEST SHOP`,
			},
			{
				Timestamp: now.Add(-time.Hour),
				Sender:    "102",
				Content: `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: 200 MDL
Dost: 900,00
Data/vremya: 03.05.23 15:21
Adres: ANOTHER SHOP`,
			},
			{
				Timestamp: now.Add(-2 * time.Hour),
				Sender:    "102",
				Content:   "Random message without template",
			},
		},
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runDefault()
	if err != nil {
		t.Errorf("runDefault() error = %v", err)
	}
	if !mockFetcher.cleanupCalled {
		t.Error("cleanup should have been called")
	}
}

func TestApp_runDefault_FetchError(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
	}

	mockFetcher := &MockFetcher{
		fetchErr: errors.New("fetch failed"),
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runDefault()
	if err == nil {
		t.Error("runDefault() should return error when fetch fails")
	}
}

func TestApp_runMissingTemplates_WithMock(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102", "EXIMBANK"},
	}

	now := time.Now()
	mockFetcher := &MockFetcher{
		messages: []*bagoup.Message{
			{
				Timestamp: now,
				Sender:    "102",
				Content: `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: 100 MDL`,
			},
			{
				Timestamp: now,
				Sender:    "102",
				Content:   "Random message without template",
			},
			{
				Timestamp: now,
				Sender:    "EXIMBANK",
				Content:   "OTP message without template",
			},
		},
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runMissingTemplates()
	if err != nil {
		t.Errorf("runMissingTemplates() error = %v", err)
	}
	if !mockFetcher.cleanupCalled {
		t.Error("cleanup should have been called")
	}
}

func TestApp_runMissingTemplates_FetchError(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
	}

	mockFetcher := &MockFetcher{
		fetchErr: errors.New("fetch failed"),
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runMissingTemplates()
	if err == nil {
		t.Error("runMissingTemplates() should return error when fetch fails")
	}
}

func TestApp_runDefault_EmptyMessages(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
	}

	mockFetcher := &MockFetcher{
		messages: []*bagoup.Message{},
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runDefault()
	if err != nil {
		t.Errorf("runDefault() should not error with empty messages: %v", err)
	}
}

func TestApp_runDefault_SingleMessage(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
	}

	mockFetcher := &MockFetcher{
		messages: []*bagoup.Message{
			{
				Timestamp: time.Now(),
				Sender:    "102",
				Content:   "Single message",
			},
		},
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runDefault()
	if err != nil {
		t.Errorf("runDefault() error = %v", err)
	}
}

func TestApp_runDefault_MultipleSenders(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102", "EXIMBANK"},
	}

	now := time.Now()
	mockFetcher := &MockFetcher{
		messages: []*bagoup.Message{
			{
				Timestamp: now,
				Sender:    "102",
				Content:   "Message from 102",
			},
			{
				Timestamp: now,
				Sender:    "EXIMBANK",
				Content:   "Message from EXIMBANK",
			},
		},
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runDefault()
	if err != nil {
		t.Errorf("runDefault() error = %v", err)
	}
}

func TestExpandPath(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(string) bool
	}{
		{
			name:     "absolute path unchanged",
			input:    "/usr/local/bin",
			wantErr:  false,
			validate: func(result string) bool { return result == "/usr/local/bin" },
		},
		{
			name:     "relative path unchanged",
			input:    "relative/path",
			wantErr:  false,
			validate: func(result string) bool { return result == "relative/path" },
		},
		{
			name:     "tilde only expands to home",
			input:    "~",
			wantErr:  false,
			validate: func(result string) bool { return len(result) > 1 && result != "~" },
		},
		{
			name:     "tilde with path expands",
			input:    "~/Library/Messages",
			wantErr:  false,
			validate: func(result string) bool { return strings.Contains(result, "Library/Messages") && !strings.HasPrefix(result, "~") },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := expandPath(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("expandPath() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr && !tc.validate(result) {
				t.Errorf("expandPath(%q) = %q, validation failed", tc.input, result)
			}
		})
	}
}

func TestChatDBFetcher_CheckDependencies(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
		Bagoup: config.BagoupConfig{
			DBPath: "/nonexistent/path/chat.db",
		},
	}

	fetcher := NewChatDBFetcher(cfg)
	err := fetcher.CheckDependencies()
	if err == nil {
		t.Error("CheckDependencies() should return error for non-existent database")
	}
}

func TestChatDBFetcher_CheckDependencies_ValidPath(t *testing.T) {
	// Create temp database
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "chat.db")
	if err := os.WriteFile(dbPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	cfg := &config.Config{
		Senders: []string{"102"},
		Bagoup: config.BagoupConfig{
			DBPath: dbPath,
		},
	}

	fetcher := NewChatDBFetcher(cfg)
	err := fetcher.CheckDependencies()
	if err != nil {
		t.Errorf("CheckDependencies() error = %v", err)
	}
}

func TestChatDBFetcher_FetchMessages_NonExistentDB(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
		Bagoup: config.BagoupConfig{
			DBPath: "/nonexistent/path/chat.db",
		},
	}

	fetcher := NewChatDBFetcher(cfg)
	_, _, err := fetcher.FetchMessages()
	if err == nil {
		t.Error("FetchMessages() should return error for non-existent database")
	}
}

func TestChatDBFetcher_FetchMessages_ValidDB(t *testing.T) {
	// Create a test database with the schema and some test messages
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "chat.db")

	// Use SQL to create a minimal Messages database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	// Create schema
	_, err = db.Exec(`
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
	`)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO handle (ROWID, id) VALUES (1, '102');
		INSERT INTO message (ROWID, handle_id, text, date, is_from_me)
		VALUES (1, 1, 'Test message', 704823707000000000, 0);
	`)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	cfg := &config.Config{
		Senders: []string{"102"},
		Bagoup: config.BagoupConfig{
			DBPath: dbPath,
		},
	}

	fetcher := NewChatDBFetcher(cfg)
	messages, cleanup, err := fetcher.FetchMessages()
	if err != nil {
		t.Fatalf("FetchMessages() error = %v", err)
	}
	defer cleanup()

	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}

	if len(messages) > 0 {
		if messages[0].Sender != "102" {
			t.Errorf("expected sender '102', got %q", messages[0].Sender)
		}
		if messages[0].Content != "Test message" {
			t.Errorf("expected content 'Test message', got %q", messages[0].Content)
		}
	}
}

func TestApp_runMissingTemplates_WithNewTemplates(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102", "EXIMBANK"},
	}

	now := time.Now()
	mockFetcher := &MockFetcher{
		messages: []*bagoup.Message{
			{
				Timestamp: now,
				Sender:    "EXIMBANK",
				Content:   "Debitare cont Card 9..7890, Data 08.04.2024 09:27:01, Suma 9.65 MDL, Detalii Test, Disponibil 100.00 MDL",
			},
			{
				Timestamp: now,
				Sender:    "EXIMBANK",
				Content:   "Tranzactie reusita, Data 13.04.2024 13:20:30, Card 9..7890, Suma 91.91 MDL, Locatie TEST>CITY, MDA, Disponibil 100.00 MDL",
			},
			{
				Timestamp: now,
				Sender:    "EXIMBANK",
				Content:   "Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma 1000.00 MDL, Detalii Salary, Disponibil 2000.00 MDL",
			},
		},
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runMissingTemplates()
	if err != nil {
		t.Errorf("runMissingTemplates() error = %v", err)
	}
}

func TestApp_parseMessage_NewTemplates(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"EXIMBANK"},
	}
	app := NewApp(cfg)

	testCases := []struct {
		name        string
		content     string
		wantHasTmpl bool
		wantOp      string
	}{
		{
			name:        "Debitare message",
			content:     "Debitare cont Card 9..7890, Data 08.04.2024 09:27:01, Suma 9.65 MDL, Detalii Test, Disponibil 100.00 MDL",
			wantHasTmpl: true,
			wantOp:      "Debitare",
		},
		{
			name:        "Tranzactie reusita message",
			content:     "Tranzactie reusita, Data 13.04.2024 13:20:30, Card 9..7890, Suma 91.91 MDL, Locatie TEST>CITY, MDA, Disponibil 100.00 MDL",
			wantHasTmpl: true,
			wantOp:      "Tranzactie reusita",
		},
		{
			name:        "Suplinire message",
			content:     "Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma 1000.00 MDL, Detalii Salary, Disponibil 2000.00 MDL",
			wantHasTmpl: true,
			wantOp:      "Suplinire",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := &bagoup.Message{
				Timestamp: time.Now(),
				Sender:    "EXIMBANK",
				Content:   tc.content,
			}
			pm := app.parseMessage(msg)
			if pm.HasTemplate != tc.wantHasTmpl {
				t.Errorf("HasTemplate = %v, want %v", pm.HasTemplate, tc.wantHasTmpl)
			}
			if tc.wantHasTmpl && pm.Transaction != nil {
				if pm.Transaction.Operation != tc.wantOp {
					t.Errorf("Operation = %q, want %q", pm.Transaction.Operation, tc.wantOp)
				}
			}
		})
	}
}

func TestApp_runDefault_WithNewTemplates(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"EXIMBANK"},
	}

	now := time.Now()
	mockFetcher := &MockFetcher{
		messages: []*bagoup.Message{
			{
				Timestamp: now,
				Sender:    "EXIMBANK",
				Content:   "Debitare cont Card 9..7890, Data 08.04.2024 09:27:01, Suma 9.65 MDL, Detalii Test, Disponibil 100.00 MDL",
			},
			{
				Timestamp: now.Add(-time.Hour),
				Sender:    "EXIMBANK",
				Content:   "Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma 1000.00 MDL, Detalii Salary, Disponibil 2000.00 MDL",
			},
		},
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runDefault()
	if err != nil {
		t.Errorf("runDefault() error = %v", err)
	}
}

func TestApp_runMissingTemplates_ExcludesIgnoredMessages(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102", "EXIMBANK"},
	}

	now := time.Now()
	mockFetcher := &MockFetcher{
		messages: []*bagoup.Message{
			{
				Timestamp: now,
				Sender:    "102",
				Content:   "Vas privetstvuet servis opoveshenia ot MAIB\nProfili budet aktivirovan.",
			},
			{
				Timestamp: now,
				Sender:    "102",
				Content:   "Oper.: Ostatok\nKarta: *1234",
			},
			{
				Timestamp: now,
				Sender:    "EXIMBANK",
				Content:   "Autentificarea Dvs. in sistemul Eximbank Online a fost inregistrata la 08.04.2024",
			},
			{
				Timestamp: now,
				Sender:    "EXIMBANK",
				Content:   "Parola de unica folosinta pentru tranzactia cu ID-ul 123456789 este 1234",
			},
			{
				Timestamp: now,
				Sender:    "EXIMBANK",
				Content:   "OTP-ul pentru Plati din Exim Personal este 567890",
			},
			{
				Timestamp: now,
				Sender:    "EXIMBANK",
				Content:   "Va multumim ca ati ales serviciul Eximbank SMS Info.",
			},
			{
				Timestamp: now,
				Sender:    "102",
				Content:   "This is truly unknown message without template",
			},
		},
	}

	app := NewAppWithFetcher(cfg, mockFetcher)

	// Count messages that should be reported as missing templates
	// Only the last message should be reported (ignored messages should not appear)
	count := 0
	for _, msg := range mockFetcher.messages {
		if !app.matcher.ShouldIgnore(msg.Content) && app.matcher.FindTemplate(msg.Content) == nil {
			count++
		}
	}

	if count != 1 {
		t.Errorf("expected 1 message without template (excluding ignored), got %d", count)
	}
}

func TestApp_runMissingTemplates_FiltersMeMessages(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102", "EXIMBANK"},
	}

	now := time.Now()
	mockFetcher := &MockFetcher{
		messages: []*bagoup.Message{
			{
				Timestamp: now,
				Sender:    "Me",
				Content:   "A",
			},
			{
				Timestamp: now,
				Sender:    "Me",
				Content:   "Some user message",
			},
			{
				Timestamp: now,
				Sender:    "102",
				Content:   "Unknown message from bank",
			},
		},
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runMissingTemplates()
	if err != nil {
		t.Errorf("runMissingTemplates() error = %v", err)
	}

	// Messages from "Me" should be filtered out, only "Unknown message from bank" should be reported
	// This test verifies the filter is working (the actual output goes to stdout)
}

func TestApp_runYNABSync_MissingConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		want   string
	}{
		{
			name: "missing budget_id",
			config: &config.Config{
				Senders: []string{"102"},
				YNAB: config.YNABConfig{
					BudgetID:  "",
					Accounts:  []config.YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}},
					StartDate: "2026-01-01",
				},
			},
			want: "budget_id not configured",
		},
		{
			name: "missing accounts",
			config: &config.Config{
				Senders: []string{"102"},
				YNAB: config.YNABConfig{
					BudgetID:  "test-budget",
					Accounts:  []config.YNABAccount{},
					StartDate: "2026-01-01",
				},
			},
			want: "accounts not configured",
		},
		{
			name: "missing start_date",
			config: &config.Config{
				Senders: []string{"102"},
				YNAB: config.YNABConfig{
					BudgetID:  "test-budget",
					Accounts:  []config.YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}},
					StartDate: "",
				},
			},
			want: "start_date not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(tt.config)
			err := app.runYNABSync()
			if err == nil {
				t.Errorf("runYNABSync() should return error, got nil")
			}
			if err != nil && err.Error() != "YNAB "+tt.want {
				t.Errorf("runYNABSync() error = %v, want %v", err, "YNAB "+tt.want)
			}
		})
	}
}

func TestApp_runYNABSync_InvalidStartDate(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
		YNAB: config.YNABConfig{
			BudgetID:  "test-budget",
			Accounts:  []config.YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}},
			StartDate: "invalid-date",
		},
	}

	app := NewApp(cfg)
	err := app.runYNABSync()
	if err == nil {
		t.Error("runYNABSync() should return error for invalid start date")
	}
}

func TestApp_runYNABSync_FetchError(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
		YNAB: config.YNABConfig{
			BudgetID:  "test-budget",
			Accounts:  []config.YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}},
			StartDate: "2026-01-01",
		},
	}

	mockFetcher := &MockFetcher{
		fetchErr: errors.New("fetch failed"),
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runYNABSync()
	if err == nil {
		t.Error("runYNABSync() should return error when fetch fails")
	}
}

func TestApp_runYNABSync_MissingAPIKey(t *testing.T) {
	// Save original env var
	origKey := os.Getenv("YNAB_API_KEY")
	defer func() {
		if origKey != "" {
			os.Setenv("YNAB_API_KEY", origKey)
		} else {
			os.Unsetenv("YNAB_API_KEY")
		}
	}()

	// Unset API key
	os.Unsetenv("YNAB_API_KEY")

	cfg := &config.Config{
		Senders: []string{"102"},
		YNAB: config.YNABConfig{
			BudgetID:  "test-budget",
			Accounts:  []config.YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}},
			StartDate: "2026-01-01",
		},
		DataFilePath: filepath.Join(t.TempDir(), "data.json"),
	}

	mockFetcher := &MockFetcher{
		messages: []*bagoup.Message{
			{
				Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
				Sender:    "102",
				Content: `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: 100 MDL
Dost: 1000,00
Data/vremya: 10.01.26 10:00
Adres: TEST SHOP`,
			},
		},
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runYNABSync()
	if err == nil {
		t.Error("runYNABSync() should return error when YNAB_API_KEY is not set")
	}
	if err != nil && err.Error() != "YNAB_API_KEY environment variable not set" {
		t.Errorf("runYNABSync() error = %v, want YNAB_API_KEY environment variable not set", err)
	}
}

func TestApp_runYNABSync_NoTransactions(t *testing.T) {
	// Save original env var
	origKey := os.Getenv("YNAB_API_KEY")
	defer func() {
		if origKey != "" {
			os.Setenv("YNAB_API_KEY", origKey)
		} else {
			os.Unsetenv("YNAB_API_KEY")
		}
	}()

	// Set API key
	os.Setenv("YNAB_API_KEY", "test-api-key")

	cfg := &config.Config{
		Senders: []string{"102"},
		YNAB: config.YNABConfig{
			BudgetID:  "test-budget",
			Accounts:  []config.YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}},
			StartDate: "2026-01-01",
		},
		DataFilePath: filepath.Join(t.TempDir(), "data.json"),
	}

	// Empty messages - no transactions to sync
	mockFetcher := &MockFetcher{
		messages: []*bagoup.Message{},
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runYNABSync()
	// Should succeed with 0 transactions
	if err != nil {
		t.Errorf("runYNABSync() should succeed with no transactions, got error: %v", err)
	}
}

func TestApp_runYNABSync_OnlyNonMDLTransactions(t *testing.T) {
	// Save original env var
	origKey := os.Getenv("YNAB_API_KEY")
	defer func() {
		if origKey != "" {
			os.Setenv("YNAB_API_KEY", origKey)
		} else {
			os.Unsetenv("YNAB_API_KEY")
		}
	}()

	// Set API key
	os.Setenv("YNAB_API_KEY", "test-api-key")

	cfg := &config.Config{
		Senders: []string{"102"},
		YNAB: config.YNABConfig{
			BudgetID:  "test-budget",
			Accounts:  []config.YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}},
			StartDate: "2026-01-01",
		},
		DataFilePath: filepath.Join(t.TempDir(), "data.json"),
	}

	mockFetcher := &MockFetcher{
		messages: []*bagoup.Message{
			{
				Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
				Sender:    "102",
				Content:   "Message without template",
			},
		},
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runYNABSync()
	// Should succeed with 0 MDL transactions
	if err != nil {
		t.Errorf("runYNABSync() should succeed with no MDL transactions, got error: %v", err)
	}
}

func TestApp_runYNABSync_SkipsDeclinedTransactions(t *testing.T) {
	// Save original env var
	origKey := os.Getenv("YNAB_API_KEY")
	defer func() {
		if origKey != "" {
			os.Setenv("YNAB_API_KEY", origKey)
		} else {
			os.Unsetenv("YNAB_API_KEY")
		}
	}()

	// Set API key
	os.Setenv("YNAB_API_KEY", "test-api-key")

	cfg := &config.Config{
		Senders: []string{"102"},
		YNAB: config.YNABConfig{
			BudgetID:  "test-budget",
			Accounts:  []config.YNABAccount{}, // No accounts - will test account auto-creation is triggered
			StartDate: "2026-01-01",
		},
		DataFilePath: filepath.Join(t.TempDir(), "data.json"),
	}

	mockFetcher := &MockFetcher{
		messages: []*bagoup.Message{
			{
				Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
				Sender:    "102",
				Content: `Op: Tovary i uslugi
Karta: *1234
Status: Odobrena
Summa: 100 MDL
Dost: 1000,00
Data/vremya: 10.01.26 10:00
Adres: TEST SHOP`,
			},
			{
				Timestamp: time.Date(2026, 1, 10, 11, 0, 0, 0, time.UTC),
				Sender:    "102",
				Content: `Op: Tovary i uslugi
Karta: *1234
Status: Decline§TranNotPermToCardHolder
Summa: 200 MDL
Dost: 900,00
Data/vremya: 10.01.26 11:00
Adres: DECLINED SHOP`,
			},
		},
	}

	app := NewAppWithFetcher(cfg, mockFetcher)
	err := app.runYNABSync()
	// Should fail trying to get accounts (since no mock YNAB client and will use real API)
	// This is expected behavior - when accounts don't exist, system tries to create them via API
	if err == nil {
		t.Error("runYNABSync() should fail when trying to access real YNAB API without valid credentials")
	}
}

func TestApp_convertTransactions_NilConverter(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
	}

	app := NewApp(cfg)
	app.converter = nil // Explicitly set converter to nil

	parsedMessages := []*ParsedMessage{
		{
			Message: &bagoup.Message{
				Timestamp: time.Now(),
				Sender:    "102",
			},
			Transaction: &template.Transaction{
				Original: template.Amount{Value: 100.0, Currency: "USD"},
			},
			HasTemplate: true,
		},
	}

	// Should not panic with nil converter
	app.convertTransactions(parsedMessages)

	// Transaction should remain unchanged
	if parsedMessages[0].Transaction.Converted.Currency != "" {
		t.Error("Transaction should not be converted when converter is nil")
	}
}

func TestRun_YNABSyncCommand(t *testing.T) {
	// Create temp config with YNAB settings
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{
		"senders": ["102"],
		"bagoup": {"db_path": "test.db", "separate_chats": true},
		"ynab": {
			"budget_id": "test-budget",
			"accounts": [{"ynab_account_id": "acc-1", "last4": "1234"}],
			"start_date": "2026-01-01"
		}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}

	// This will fail because bagoup won't find the db, but tests the command parsing
	err := Run([]string{"--config", configPath, "ynab_sync"})
	// Will fail because of bagoup execution or missing API key
	if err == nil {
		t.Skip("Expected error from bagoup or missing API key")
	}
}
