package ynab

import (
	"os"
	"testing"
)

func TestNewCategoryStore(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)
	if store == nil {
		t.Error("NewCategoryStore() returned nil")
	}
}

func TestCategoryStore_GetEmpty(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)

	catID := store.Get("Unknown Payee")
	if catID != "" {
		t.Errorf("Get() = %q, want empty string for unknown payee", catID)
	}
}

func TestCategoryStore_SetAndGet(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)

	err := store.Set("Coffee Shop", "cat-123")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	catID := store.Get("Coffee Shop")
	if catID != "cat-123" {
		t.Errorf("Get() = %q, want %q", catID, "cat-123")
	}
}

func TestCategoryStore_SetBatch(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)

	mapping := map[string]string{
		"Coffee Shop": "cat-food",
		"Salary":      "cat-income",
		"Gas Station": "cat-transport",
	}

	err := store.SetBatch(mapping)
	if err != nil {
		t.Fatalf("SetBatch() error = %v", err)
	}

	if store.Get("Coffee Shop") != "cat-food" {
		t.Errorf("Get(Coffee Shop) = %q, want cat-food", store.Get("Coffee Shop"))
	}
	if store.Get("Salary") != "cat-income" {
		t.Errorf("Get(Salary) = %q, want cat-income", store.Get("Salary"))
	}
	if store.Get("Gas Station") != "cat-transport" {
		t.Errorf("Get(Gas Station) = %q, want cat-transport", store.Get("Gas Station"))
	}
}

func TestCategoryStore_All(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)

	_ = store.Set("Coffee Shop", "cat-food")
	_ = store.Set("Salary", "cat-income")

	all, err := store.All()
	if err != nil {
		t.Fatalf("All() error = %v", err)
	}

	if len(all) != 2 {
		t.Errorf("All() returned %d entries, want 2", len(all))
	}
	if all["Coffee Shop"] != "cat-food" {
		t.Errorf("all[Coffee Shop] = %q, want cat-food", all["Coffee Shop"])
	}
}

func TestCategoryStore_PersistsAcrossInstances(t *testing.T) {
	filePath := t.TempDir() + "/data.json"

	store1 := NewCategoryStore(filePath)
	_ = store1.Set("Coffee Shop", "cat-123")

	store2 := NewCategoryStore(filePath)
	catID := store2.Get("Coffee Shop")
	if catID != "cat-123" {
		t.Errorf("Get() after reload = %q, want cat-123 (not persisted)", catID)
	}
}

func TestCategoryStore_PreservesExistingDataFileKeys(t *testing.T) {
	filePath := t.TempDir() + "/data.json"

	// First create sync store data to simulate existing file
	syncStore, err := NewSyncStore(filePath)
	if err != nil {
		t.Fatalf("NewSyncStore() error = %v", err)
	}
	_ = syncStore.RecordSync(&SyncRecord{ImportID: "YNAB:test123"})
	syncStore.Close()

	// Now write category store entries
	categoryStore := NewCategoryStore(filePath)
	_ = categoryStore.Set("Coffee Shop", "cat-123")

	// Verify sync store data is still intact
	syncStore2, err := NewSyncStore(filePath)
	if err != nil {
		t.Fatalf("NewSyncStore() after category write error = %v", err)
	}
	records, err := syncStore2.GetAllSynced()
	if err != nil {
		t.Fatalf("GetAllSynced() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("GetAllSynced() returned %d records, want 1 (data lost)", len(records))
	}
}

func TestCategoryStore_UpdateExistingPayee(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)

	_ = store.Set("Coffee Shop", "cat-old")
	_ = store.Set("Coffee Shop", "cat-new")

	catID := store.Get("Coffee Shop")
	if catID != "cat-new" {
		t.Errorf("Get() = %q, want cat-new (should update)", catID)
	}
}

func TestCategoryStore_GetInvalidFile(t *testing.T) {
	filePath := t.TempDir() + "/invalid.json"
	// Write invalid JSON to the file
	if err := os.WriteFile(filePath, []byte("invalid json"), 0600); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	store := NewCategoryStore(filePath)
	catID := store.Get("payee")
	if catID != "" {
		t.Errorf("Get() = %q, want empty for invalid file", catID)
	}
}

func TestCategoryStore_SetInvalidFile(t *testing.T) {
	filePath := t.TempDir() + "/invalid.json"
	if err := os.WriteFile(filePath, []byte("invalid json"), 0600); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	store := NewCategoryStore(filePath)
	err := store.Set("payee", "cat-1")
	if err == nil {
		t.Error("Set() should return error for invalid file")
	}
}

func TestCategoryStore_SetBatchInvalidFile(t *testing.T) {
	filePath := t.TempDir() + "/invalid.json"
	if err := os.WriteFile(filePath, []byte("invalid json"), 0600); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	store := NewCategoryStore(filePath)
	err := store.SetBatch(map[string]string{"payee": "cat-1"})
	if err == nil {
		t.Error("SetBatch() should return error for invalid file")
	}
}

func TestCategoryStore_AllInvalidFile(t *testing.T) {
	filePath := t.TempDir() + "/invalid.json"
	if err := os.WriteFile(filePath, []byte("invalid json"), 0600); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	store := NewCategoryStore(filePath)
	_, err := store.All()
	if err == nil {
		t.Error("All() should return error for invalid file")
	}
}
