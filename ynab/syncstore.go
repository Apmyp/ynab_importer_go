package ynab

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

type SyncStore struct {
	filePath string
	mu       sync.RWMutex
}

type dataFile struct {
	Rates                  []interface{}     `json:"rates"`
	YNABSyncedTransactions []SyncRecord      `json:"ynab_synced_transactions"`
	PayeeCategories        map[string]string `json:"payee_categories,omitempty"`
}

func NewSyncStore(filePath string) (*SyncStore, error) {
	store := &SyncStore{
		filePath: filePath,
	}

	if err := store.ensureFileExists(); err != nil {
		return nil, err
	}

	return store, nil
}

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

func (s *SyncStore) writeFile(data *dataFile) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, content, 0600)
}

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

func (s *SyncStore) RecordSync(record *SyncRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.readFile()
	if err != nil {
		return err
	}

	for i, existing := range data.YNABSyncedTransactions {
		if existing.ImportID == record.ImportID {
			data.YNABSyncedTransactions[i] = *record
			return s.writeFile(data)
		}
	}

	data.YNABSyncedTransactions = append(data.YNABSyncedTransactions, *record)
	return s.writeFile(data)
}

func (s *SyncStore) GetAllSynced() ([]SyncRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.readFile()
	if err != nil {
		return nil, err
	}

	return data.YNABSyncedTransactions, nil
}

func (s *SyncStore) Close() error {
	return nil
}

var ErrNotSynced = errors.New("transaction not synced")

func (s *SyncStore) DeleteSyncedOnOrAfter(date time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.readFile()
	if err != nil {
		return 0, err
	}

	var kept []SyncRecord
	deleted := 0
	for _, record := range data.YNABSyncedTransactions {
		if record.TransactionDate == "" {
			kept = append(kept, record)
			continue
		}
		txDate, err := time.Parse("2006-01-02", record.TransactionDate)
		if err != nil {
			kept = append(kept, record)
			continue
		}
		if txDate.Before(date) {
			kept = append(kept, record)
		} else {
			deleted++
		}
	}

	data.YNABSyncedTransactions = kept
	if kept == nil {
		data.YNABSyncedTransactions = []SyncRecord{}
	}
	if err := s.writeFile(data); err != nil {
		return 0, err
	}
	return deleted, nil
}

func (s *SyncStore) DeleteAllSynced() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.readFile()
	if err != nil {
		return 0, err
	}

	deleted := len(data.YNABSyncedTransactions)
	data.YNABSyncedTransactions = []SyncRecord{}
	if err := s.writeFile(data); err != nil {
		return 0, err
	}
	return deleted, nil
}
