// Package config handles loading application configuration
package config

import (
	"encoding/json"
	"os"
)

// BagoupConfig holds bagoup command configuration
type BagoupConfig struct {
	DBPath        string `json:"db_path"`
	SeparateChats bool   `json:"separate_chats"`
}

// YNABAccount maps a card's last 4 digits to a YNAB account ID
type YNABAccount struct {
	YNABAccountID string `json:"ynab_account_id"`
	Last4         string `json:"last4"`
}

// YNABConfig holds YNAB integration configuration
type YNABConfig struct {
	BudgetID  string        `json:"budget_id"`
	Accounts  []YNABAccount `json:"accounts"`
	StartDate string        `json:"start_date"` // YYYY-MM-DD format
}

// Config represents the application configuration
type Config struct {
	Senders         []string     `json:"senders"`
	Bagoup          BagoupConfig `json:"bagoup"`
	DefaultCurrency string       `json:"default_currency"`
	DataFilePath    string       `json:"data_file_path"`
	YNAB            YNABConfig   `json:"ynab"`
}

// Load reads and parses a configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.DefaultCurrency == "" {
		cfg.DefaultCurrency = "MDL"
	}
	if cfg.DataFilePath == "" {
		cfg.DataFilePath = "ynab_importer_go_data.json"
	}

	return &cfg, nil
}

// GetSenders returns the list of senders to filter
func (c *Config) GetSenders() []string {
	return c.Senders
}

// Save writes the configuration to a file
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
