package ynab

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

type SyncStore struct {
	filePath string
	mu       sync.RWMutex
}

type dataFile struct {
	Rates                  []interface{} `json:"rates"`
	YNABSyncedTransactions []SyncRecord  `json:"ynab_synced_transactions"`
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
