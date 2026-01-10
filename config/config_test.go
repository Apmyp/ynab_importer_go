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
