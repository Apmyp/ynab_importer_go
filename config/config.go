package config

import (
	"encoding/json"
	"os"
)

type YNABAccount struct {
	YNABAccountID string `json:"ynab_account_id"`
	Last4         string `json:"last4"`
}

type YNABConfig struct {
	BudgetID  string        `json:"budget_id"`
	Accounts  []YNABAccount `json:"accounts"`
	StartDate string        `json:"start_date"`
}

type Config struct {
	Senders         []string   `json:"senders"`
	DBPath          string     `json:"db_path"`
	DefaultCurrency string     `json:"default_currency"`
	DataFilePath    string     `json:"data_file_path"`
	YNAB            YNABConfig `json:"ynab"`
}

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

func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
