package exchangerate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStore_CreatesDatabase(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Error("NewStore() returned nil store")
	}
}

func TestStore_SaveAndGetRate(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	rate := &Rate{
		Date:     date,
		Currency: "USD",
		Value:    18.5,
	}

	err = store.SaveRate(rate)
	if err != nil {
		t.Fatalf("SaveRate() error = %v", err)
	}

	retrieved, err := store.GetRate(date, "USD")
	if err != nil {
		t.Fatalf("GetRate() error = %v", err)
	}

	if retrieved.Currency != "USD" {
		t.Errorf("expected currency USD, got %s", retrieved.Currency)
	}
	if retrieved.Value != 18.5 {
		t.Errorf("expected value 18.5, got %f", retrieved.Value)
	}
}

func TestStore_GetRate_NotFound(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err = store.GetRate(date, "EUR")
	if err == nil {
		t.Error("GetRate() should return error for non-existent rate")
	}
}

func TestStore_SaveRate_Duplicate(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	rate1 := &Rate{
		Date:     date,
		Currency: "USD",
		Value:    18.5,
	}
	rate2 := &Rate{
		Date:     date,
		Currency: "USD",
		Value:    19.0,
	}

	err = store.SaveRate(rate1)
	if err != nil {
		t.Fatalf("SaveRate() first error = %v", err)
	}

	err = store.SaveRate(rate2)
	if err != nil {
		t.Fatalf("SaveRate() second error = %v", err)
	}

	retrieved, err := store.GetRate(date, "USD")
	if err != nil {
		t.Fatalf("GetRate() error = %v", err)
	}
	if retrieved.Value != 19.0 {
		t.Errorf("expected updated value 19.0, got %f", retrieved.Value)
	}
}

func TestNewStore_InvalidPath(t *testing.T) {
	// Try to create store in non-existent directory without permissions
	_, err := NewStore("/nonexistent/invalid/path/test.db")
	if err == nil {
		t.Error("NewStore() should return error for invalid path")
	}
}

func TestStore_GetRate_InvalidDateFormat(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	// Manually write invalid JSON to test error handling
	content := []byte(`{"rates": [{"date": "invalid-date", "currency": "USD", "value": 18.5}]}`)
	os.WriteFile(dbPath, content, 0644)

	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err = store.GetRate(date, "USD")
	// Should return ErrRateNotFound since invalid date won't match
	if err != ErrRateNotFound {
		t.Errorf("expected ErrRateNotFound, got %v", err)
	}
}

func TestStore_ReadFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create file with invalid JSON
	os.WriteFile(dbPath, []byte("invalid json{{{"), 0644)

	store := &Store{filePath: dbPath}
	_, err := store.readFile()
	if err == nil {
		t.Error("readFile() should return error for invalid JSON")
	}
}

func TestStore_SaveRate_ReadError(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create file with invalid JSON
	os.WriteFile(dbPath, []byte("invalid json"), 0644)

	store := &Store{filePath: dbPath}
	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	rate := &Rate{
		Date:     date,
		Currency: "USD",
		Value:    18.5,
	}

	err := store.SaveRate(rate)
	if err == nil {
		t.Error("SaveRate() should return error when file has invalid JSON")
	}
}

func TestStore_GetRate_ReadError(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create file with invalid JSON
	os.WriteFile(dbPath, []byte("not valid json at all"), 0644)

	store := &Store{filePath: dbPath}
	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err := store.GetRate(date, "USD")
	if err == nil {
		t.Error("GetRate() should return error when file has invalid JSON")
	}
}
