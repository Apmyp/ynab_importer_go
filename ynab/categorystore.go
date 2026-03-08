package ynab

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type CategoryStore struct {
	filePath string
	mu       sync.RWMutex
}

func NewCategoryStore(filePath string) *CategoryStore {
	return &CategoryStore{filePath: filePath}
}

func (s *CategoryStore) readFile() (*dataFile, error) {
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return &dataFile{
			Rates:                  []interface{}{},
			YNABSyncedTransactions: []SyncRecord{},
			PayeeCategories:        map[string]string{},
		}, nil
	}

	content, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	var data dataFile
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	if data.PayeeCategories == nil {
		data.PayeeCategories = map[string]string{}
	}

	return &data, nil
}

func (s *CategoryStore) writeFile(data *dataFile) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, content, 0600)
}

func (s *CategoryStore) Get(payeeName string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.readFile()
	if err != nil {
		return ""
	}
	return data.PayeeCategories[payeeName]
}

func (s *CategoryStore) Set(payeeName, categoryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.readFile()
	if err != nil {
		return err
	}

	data.PayeeCategories[payeeName] = categoryID
	return s.writeFile(data)
}

func (s *CategoryStore) SetBatch(mapping map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.readFile()
	if err != nil {
		return err
	}

	for payee, catID := range mapping {
		data.PayeeCategories[payee] = catID
	}
	return s.writeFile(data)
}

func (s *CategoryStore) IsSeededRecently(ttl time.Duration) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.readFile()
	if err != nil {
		return false, err
	}

	if data.CategoriesSeededAt == nil {
		return false, nil
	}

	return time.Since(*data.CategoriesSeededAt) < ttl, nil
}

func (s *CategoryStore) MarkSeeded() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.readFile()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	data.CategoriesSeededAt = &now
	return s.writeFile(data)
}

func (s *CategoryStore) All() (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.readFile()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(data.PayeeCategories))
	for k, v := range data.PayeeCategories {
		result[k] = v
	}
	return result, nil
}
