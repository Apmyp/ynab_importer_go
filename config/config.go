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

// Config represents the application configuration
type Config struct {
	Senders []string     `json:"senders"`
	Bagoup  BagoupConfig `json:"bagoup"`
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

	return &cfg, nil
}

// GetSenders returns the list of senders to filter
func (c *Config) GetSenders() []string {
	return c.Senders
}
