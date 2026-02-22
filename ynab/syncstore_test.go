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

func TestSyncStore_DeleteSyncedOnOrAfter_DeletesMatchingRecords(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewSyncStore(storePath)
	if err != nil {
		t.Fatalf("NewSyncStore() error = %v", err)
	}
	defer store.Close()

	records := []*SyncRecord{
		{ImportID: "ID1", SyncedAt: time.Now().UTC(), TransactionDate: "2026-01-01"},
		{ImportID: "ID2", SyncedAt: time.Now().UTC(), TransactionDate: "2026-01-15"},
		{ImportID: "ID3", SyncedAt: time.Now().UTC(), TransactionDate: "2026-02-01"},
		{ImportID: "ID4", SyncedAt: time.Now().UTC(), TransactionDate: "2025-12-31"},
	}
	for _, r := range records {
		if err := store.RecordSync(r); err != nil {
			t.Fatalf("RecordSync() error = %v", err)
		}
	}

	cutoff, _ := time.Parse("2006-01-02", "2026-01-15")
	n, err := store.DeleteSyncedOnOrAfter(cutoff)
	if err != nil {
		t.Fatalf("DeleteSyncedOnOrAfter() error = %v", err)
	}
	if n != 2 {
		t.Errorf("DeleteSyncedOnOrAfter() deleted %d records, want 2", n)
	}

	synced, _ := store.GetAllSynced()
	if len(synced) != 2 {
		t.Errorf("GetAllSynced() returned %d records, want 2", len(synced))
	}
}

func TestSyncStore_DeleteSyncedOnOrAfter_NoMatchingRecords(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewSyncStore(storePath)
	if err != nil {
		t.Fatalf("NewSyncStore() error = %v", err)
	}
	defer store.Close()

	records := []*SyncRecord{
		{ImportID: "ID1", SyncedAt: time.Now().UTC(), TransactionDate: "2025-12-01"},
		{ImportID: "ID2", SyncedAt: time.Now().UTC(), TransactionDate: "2025-12-31"},
	}
	for _, r := range records {
		if err := store.RecordSync(r); err != nil {
			t.Fatalf("RecordSync() error = %v", err)
		}
	}

	cutoff, _ := time.Parse("2006-01-02", "2026-01-01")
	n, err := store.DeleteSyncedOnOrAfter(cutoff)
	if err != nil {
		t.Fatalf("DeleteSyncedOnOrAfter() error = %v", err)
	}
	if n != 0 {
		t.Errorf("DeleteSyncedOnOrAfter() deleted %d records, want 0", n)
	}

	synced, _ := store.GetAllSynced()
	if len(synced) != 2 {
		t.Errorf("GetAllSynced() returned %d records, want 2", len(synced))
	}
}

func TestSyncStore_DeleteSyncedOnOrAfter_EmptyTransactionDateSkipped(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewSyncStore(storePath)
	if err != nil {
		t.Fatalf("NewSyncStore() error = %v", err)
	}
	defer store.Close()

	records := []*SyncRecord{
		{ImportID: "ID1", SyncedAt: time.Now().UTC(), TransactionDate: ""},
		{ImportID: "ID2", SyncedAt: time.Now().UTC(), TransactionDate: "2026-01-15"},
	}
	for _, r := range records {
		if err := store.RecordSync(r); err != nil {
			t.Fatalf("RecordSync() error = %v", err)
		}
	}

	cutoff, _ := time.Parse("2006-01-02", "2026-01-01")
	n, err := store.DeleteSyncedOnOrAfter(cutoff)
	if err != nil {
		t.Fatalf("DeleteSyncedOnOrAfter() error = %v", err)
	}
	if n != 1 {
		t.Errorf("DeleteSyncedOnOrAfter() deleted %d records, want 1", n)
	}
}

func TestSyncStore_DeleteAllSynced_RemovesAll(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewSyncStore(storePath)
	if err != nil {
		t.Fatalf("NewSyncStore() error = %v", err)
	}
	defer store.Close()

	records := []*SyncRecord{
		{ImportID: "ID1", SyncedAt: time.Now().UTC()},
		{ImportID: "ID2", SyncedAt: time.Now().UTC()},
		{ImportID: "ID3", SyncedAt: time.Now().UTC()},
	}
	for _, r := range records {
		if err := store.RecordSync(r); err != nil {
			t.Fatalf("RecordSync() error = %v", err)
		}
	}

	n, err := store.DeleteAllSynced()
	if err != nil {
		t.Fatalf("DeleteAllSynced() error = %v", err)
	}
	if n != 3 {
		t.Errorf("DeleteAllSynced() deleted %d records, want 3", n)
	}

	synced, _ := store.GetAllSynced()
	if len(synced) != 0 {
		t.Errorf("GetAllSynced() returned %d records after DeleteAllSynced(), want 0", len(synced))
	}
}

func TestSyncStore_DeleteAllSynced_EmptyStore(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "data.json")
	store, err := NewSyncStore(storePath)
	if err != nil {
		t.Fatalf("NewSyncStore() error = %v", err)
	}
	defer store.Close()

	n, err := store.DeleteAllSynced()
	if err != nil {
		t.Fatalf("DeleteAllSynced() error = %v", err)
	}
	if n != 0 {
		t.Errorf("DeleteAllSynced() deleted %d records, want 0", n)
	}
}
