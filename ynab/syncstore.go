package ynab

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

// SyncStore manages the persistence of synced transaction records
type SyncStore struct {
	filePath string
	mu       sync.RWMutex
}

// dataFile represents the structure of the data file
type dataFile struct {
	Rates                  []interface{} `json:"rates"`                    // Preserve existing rates
	YNABSyncedTransactions []SyncRecord  `json:"ynab_synced_transactions"` // Our records
}

// NewSyncStore creates a new SyncStore
func NewSyncStore(filePath string) (*SyncStore, error) {
	store := &SyncStore{
		filePath: filePath,
	}

	if err := store.ensureFileExists(); err != nil {
		return nil, err
	}

	return store, nil
}

// ensureFileExists creates an empty data file if it doesn't exist
func (s *SyncStore) ensureFileExists() error {
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		data := dataFile{
			Rates:                  []interface{}{},
			YNABSyncedTransactions: []SyncRecord{},
		}
		return s.writeFile(&data)
	}
	return nil
}

// readFile reads and unmarshals the data file
func (s *SyncStore) readFile() (*dataFile, error) {
	content, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	var data dataFile
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// writeFile marshals and writes the data file
func (s *SyncStore) writeFile(data *dataFile) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, content, 0644)
}

// IsSynced checks if a transaction with the given import_id has been synced
func (s *SyncStore) IsSynced(importID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.readFile()
	if err != nil {
		return false, err
	}

	for _, record := range data.YNABSyncedTransactions {
		if record.ImportID == importID {
			return true, nil
		}
	}

	return false, nil
}

// RecordSync records a successful sync
func (s *SyncStore) RecordSync(record *SyncRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.readFile()
	if err != nil {
		return err
	}

	// Check if already exists
	for i, existing := range data.YNABSyncedTransactions {
		if existing.ImportID == record.ImportID {
			// Update existing record
			data.YNABSyncedTransactions[i] = *record
			return s.writeFile(data)
		}
	}

	// Append new record
	data.YNABSyncedTransactions = append(data.YNABSyncedTransactions, *record)
	return s.writeFile(data)
}

// GetAllSynced returns all synced records
func (s *SyncStore) GetAllSynced() ([]SyncRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.readFile()
	if err != nil {
		return nil, err
	}

	return data.YNABSyncedTransactions, nil
}

// Close is a no-op but matches the interface pattern
func (s *SyncStore) Close() error {
	return nil
}

// ErrNotSynced is returned when a transaction is not found in the sync records
var ErrNotSynced = errors.New("transaction not synced")
