package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	// Create temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{
  "senders": ["102", "EXIMBANK"],
  "bagoup": {
    "db_path": "~/Library/Messages/chat.db",
    "separate_chats": true
  }
}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Senders) != 2 {
		t.Errorf("expected 2 senders, got %d", len(cfg.Senders))
	}
	if cfg.Senders[0] != "102" {
		t.Errorf("expected first sender to be '102', got %q", cfg.Senders[0])
	}
	if cfg.Senders[1] != "EXIMBANK" {
		t.Errorf("expected second sender to be 'EXIMBANK', got %q", cfg.Senders[1])
	}
	if cfg.Bagoup.DBPath != "~/Library/Messages/chat.db" {
		t.Errorf("expected db_path '~/Library/Messages/chat.db', got %q", cfg.Bagoup.DBPath)
	}
	if !cfg.Bagoup.SeparateChats {
		t.Error("expected separate_chats to be true")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Load() should return error for non-existent file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{invalid json`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

func TestConfig_GetSenders(t *testing.T) {
	cfg := &Config{
		Senders: []string{"102", "EXIMBANK"},
	}

	senders := cfg.GetSenders()
	if len(senders) != 2 {
		t.Errorf("expected 2 senders, got %d", len(senders))
	}
}

func TestLoad_DefaultCurrencyAndDataFilePath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{
  "senders": ["102"],
  "bagoup": {
    "db_path": "~/Library/Messages/chat.db",
    "separate_chats": true
  },
  "default_currency": "USD",
  "data_file_path": "custom_data.json"
}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DefaultCurrency != "USD" {
		t.Errorf("expected default_currency 'USD', got %q", cfg.DefaultCurrency)
	}
	if cfg.DataFilePath != "custom_data.json" {
		t.Errorf("expected data_file_path 'custom_data.json', got %q", cfg.DataFilePath)
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{
  "senders": ["102"],
  "bagoup": {
    "db_path": "~/Library/Messages/chat.db",
    "separate_chats": true
  }
}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DefaultCurrency != "MDL" {
		t.Errorf("expected default currency 'MDL', got %q", cfg.DefaultCurrency)
	}
	if cfg.DataFilePath != "ynab_importer_go_data.json" {
		t.Errorf("expected default data_file_path 'ynab_importer_go_data.json', got %q", cfg.DataFilePath)
	}
}

func TestLoad_WithYNABConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{
  "senders": ["102"],
  "bagoup": {
    "db_path": "~/Library/Messages/chat.db"
  },
  "ynab": {
    "budget_id": "test-budget-id",
    "start_date": "2026-01-01",
    "accounts": [
      {"ynab_account_id": "account-1", "last4": "1234"},
      {"ynab_account_id": "account-2", "last4": "5678"}
    ]
  }
}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.YNAB.BudgetID != "test-budget-id" {
		t.Errorf("BudgetID = %v, want test-budget-id", cfg.YNAB.BudgetID)
	}

	if cfg.YNAB.StartDate != "2026-01-01" {
		t.Errorf("StartDate = %v, want 2026-01-01", cfg.YNAB.StartDate)
	}

	if len(cfg.YNAB.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(cfg.YNAB.Accounts))
	}

	if cfg.YNAB.Accounts[0].YNABAccountID != "account-1" {
		t.Errorf("Account 0 ID = %v, want account-1", cfg.YNAB.Accounts[0].YNABAccountID)
	}

	if cfg.YNAB.Accounts[0].Last4 != "1234" {
		t.Errorf("Account 0 Last4 = %v, want 1234", cfg.YNAB.Accounts[0].Last4)
	}
}

func TestConfig_Save(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	cfg := &Config{
		Senders: []string{"102", "EXIMBANK"},
		Bagoup: BagoupConfig{
			DBPath:        "~/Library/Messages/chat.db",
			SeparateChats: true,
		},
		DefaultCurrency: "MDL",
		DataFilePath:    "ynab_importer_go_data.json",
		YNAB: YNABConfig{
			BudgetID:  "test-budget-id",
			StartDate: "2026-01-01",
			Accounts: []YNABAccount{
				{YNABAccountID: "account-1", Last4: "1234"},
				{YNABAccountID: "account-2", Last4: "5678"},
			},
		},
	}

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load and verify content
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() after Save() error = %v", err)
	}

	// Verify all fields match
	if len(loadedCfg.Senders) != len(cfg.Senders) {
		t.Errorf("Senders length = %d, want %d", len(loadedCfg.Senders), len(cfg.Senders))
	}
	if loadedCfg.DefaultCurrency != cfg.DefaultCurrency {
		t.Errorf("DefaultCurrency = %v, want %v", loadedCfg.DefaultCurrency, cfg.DefaultCurrency)
	}
	if loadedCfg.YNAB.BudgetID != cfg.YNAB.BudgetID {
		t.Errorf("BudgetID = %v, want %v", loadedCfg.YNAB.BudgetID, cfg.YNAB.BudgetID)
	}
	if len(loadedCfg.YNAB.Accounts) != len(cfg.YNAB.Accounts) {
		t.Errorf("Accounts length = %d, want %d", len(loadedCfg.YNAB.Accounts), len(cfg.YNAB.Accounts))
	}
}

func TestConfig_Save_InvalidPath(t *testing.T) {
	cfg := &Config{
		Senders: []string{"102"},
	}

	// Try to save to non-existent directory without creating it
	err := cfg.Save("/nonexistent/directory/config.json")
	if err == nil {
		t.Error("Save() should return error for invalid path")
	}
}
