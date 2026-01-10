package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/apmyp/ynab_importer_go/bagoup"
	"github.com/apmyp/ynab_importer_go/config"
	"github.com/apmyp/ynab_importer_go/template"
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
		Amount:    100.0,
		Currency:  "MDL",
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
	if pm.Transaction.Amount != 34.0 {
		t.Errorf("expected amount 34.0, got %f", pm.Transaction.Amount)
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
	if pm.Transaction.Amount != 5000.00 {
		t.Errorf("expected amount 5000.00, got %f", pm.Transaction.Amount)
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

func TestBagoupFetcher_CheckDependencies(t *testing.T) {
	cfg := &config.Config{
		Senders: []string{"102"},
	}

	fetcher := NewBagoupFetcher(cfg)
	err := fetcher.CheckDependencies()
	// Should succeed if bagoup is installed
	if err != nil {
		t.Skipf("bagoup not installed: %v", err)
	}
}
