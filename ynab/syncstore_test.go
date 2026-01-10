package ynab

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSyncStore(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")

	store, err := NewSyncStore(storePath)
	if err != nil {
		t.Fatalf("NewSyncStore() error = %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Error("NewSyncStore() returned nil store")
	}

	// File should be created
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		t.Error("NewSyncStore() did not create file")
	}
}

func TestSyncStore_IsSynced(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewSyncStore(storePath)
	if err != nil {
		t.Fatalf("NewSyncStore() error = %v", err)
	}
	defer store.Close()

	// Should return false for new import_id
	synced, err := store.IsSynced("YNAB:1234:2026-01-10:12345")
	if err != nil {
		t.Fatalf("IsSynced() error = %v", err)
	}
	if synced {
		t.Error("IsSynced() should return false for new import_id")
	}
}

func TestSyncStore_RecordSync(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewSyncStore(storePath)
	if err != nil {
		t.Fatalf("NewSyncStore() error = %v", err)
	}
	defer store.Close()

	record := &SyncRecord{
		ImportID: "YNAB:1234:2026-01-10:12345",
		SyncedAt: time.Now().UTC(),
	}

	if err := store.RecordSync(record); err != nil {
		t.Fatalf("RecordSync() error = %v", err)
	}

	// Should now return true
	synced, err := store.IsSynced(record.ImportID)
	if err != nil {
		t.Fatalf("IsSynced() error = %v", err)
	}
	if !synced {
		t.Error("IsSynced() should return true after RecordSync()")
	}
}

func TestSyncStore_GetAllSynced(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewSyncStore(storePath)
	if err != nil {
		t.Fatalf("NewSyncStore() error = %v", err)
	}
	defer store.Close()

	// Record multiple syncs
	records := []*SyncRecord{
		{ImportID: "ID1", SyncedAt: time.Now().UTC()},
		{ImportID: "ID2", SyncedAt: time.Now().UTC()},
		{ImportID: "ID3", SyncedAt: time.Now().UTC()},
	}

	for _, record := range records {
		if err := store.RecordSync(record); err != nil {
			t.Fatalf("RecordSync() error = %v", err)
		}
	}

	// Get all synced
	synced, err := store.GetAllSynced()
	if err != nil {
		t.Fatalf("GetAllSynced() error = %v", err)
	}

	if len(synced) != 3 {
		t.Errorf("GetAllSynced() returned %d records, want 3", len(synced))
	}
}

func TestSyncStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")

	// Create store and record sync
	store1, err := NewSyncStore(storePath)
	if err != nil {
		t.Fatalf("NewSyncStore() error = %v", err)
	}

	record := &SyncRecord{
		ImportID: "YNAB:1234:2026-01-10:12345",
		SyncedAt: time.Now().UTC(),
	}

	if err := store1.RecordSync(record); err != nil {
		t.Fatalf("RecordSync() error = %v", err)
	}
	store1.Close()

	// Reopen store and check if record is still there
	store2, err := NewSyncStore(storePath)
	if err != nil {
		t.Fatalf("NewSyncStore() error = %v", err)
	}
	defer store2.Close()

	synced, err := store2.IsSynced(record.ImportID)
	if err != nil {
		t.Fatalf("IsSynced() error = %v", err)
	}
	if !synced {
		t.Error("Record should persist across store instances")
	}
}

func TestSyncStore_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")

	// Write invalid JSON
	os.WriteFile(storePath, []byte("invalid json{{{"), 0644)

	store := &SyncStore{filePath: storePath}
	_, err := store.readFile()
	if err == nil {
		t.Error("readFile() should return error for invalid JSON")
	}
}
