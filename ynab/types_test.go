package ynab

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTransactionPayload_JSON(t *testing.T) {
	tx := TransactionPayload{
		AccountID: "test-account-id",
		Date:      "2026-01-10",
		Amount:    -15000,
		PayeeName: "Test Merchant",
		Memo:      "Test transaction",
		Cleared:   "cleared",
		ImportID:  "YNAB:1234:2026-01-10:12345",
	}

	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded TransactionPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.AccountID != tx.AccountID {
		t.Errorf("AccountID mismatch: got %s, want %s", decoded.AccountID, tx.AccountID)
	}
	if decoded.Amount != tx.Amount {
		t.Errorf("Amount mismatch: got %d, want %d", decoded.Amount, tx.Amount)
	}
	if decoded.ImportID != tx.ImportID {
		t.Errorf("ImportID mismatch: got %s, want %s", decoded.ImportID, tx.ImportID)
	}
}

func TestSyncRecord_JSON(t *testing.T) {
	now := time.Now().UTC()
	record := SyncRecord{
		ImportID: "YNAB:1234:2026-01-10:12345",
		SyncedAt: now,
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded SyncRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ImportID != record.ImportID {
		t.Errorf("ImportID mismatch: got %s, want %s", decoded.ImportID, record.ImportID)
	}

	// Compare times with truncation due to JSON serialization
	if !decoded.SyncedAt.Truncate(time.Second).Equal(record.SyncedAt.Truncate(time.Second)) {
		t.Errorf("SyncedAt mismatch: got %v, want %v", decoded.SyncedAt, record.SyncedAt)
	}
}

func TestYNABAccount_JSON(t *testing.T) {
	account := YNABAccount{
		YNABAccountID: "account-uuid",
		Last4:         "1234",
	}

	data, err := json.Marshal(account)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded YNABAccount
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.YNABAccountID != account.YNABAccountID {
		t.Errorf("YNABAccountID mismatch")
	}
	if decoded.Last4 != account.Last4 {
		t.Errorf("Last4 mismatch")
	}
}

func TestTransactionPayload_CategoryID_JSON(t *testing.T) {
	tx := TransactionPayload{
		AccountID:  "test-account-id",
		Date:       "2026-01-10",
		Amount:     -15000,
		CategoryID: "cat-123",
		Cleared:    "cleared",
	}

	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded TransactionPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.CategoryID != "cat-123" {
		t.Errorf("CategoryID = %q, want %q", decoded.CategoryID, "cat-123")
	}
}

func TestTransactionPayload_CategoryID_OmittedWhenEmpty(t *testing.T) {
	tx := TransactionPayload{
		AccountID: "test-account-id",
		Date:      "2026-01-10",
		Amount:    -15000,
		Cleared:   "cleared",
	}

	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Failed to unmarshal into map: %v", err)
	}

	if _, exists := m["category_id"]; exists {
		t.Error("category_id should be omitted when empty")
	}
}

func TestTransactionDetail_JSON(t *testing.T) {
	detail := TransactionDetail{
		ID:           "txn-1",
		Date:         "2026-01-10",
		PayeeName:    "Coffee Shop",
		CategoryID:   "cat-123",
		CategoryName: "Food",
		Memo:         "lunch",
		Amount:       -5000,
	}

	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded TransactionDetail
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != detail.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, detail.ID)
	}
	if decoded.PayeeName != detail.PayeeName {
		t.Errorf("PayeeName = %q, want %q", decoded.PayeeName, detail.PayeeName)
	}
	if decoded.CategoryID != detail.CategoryID {
		t.Errorf("CategoryID = %q, want %q", decoded.CategoryID, detail.CategoryID)
	}
	if decoded.CategoryName != detail.CategoryName {
		t.Errorf("CategoryName = %q, want %q", decoded.CategoryName, detail.CategoryName)
	}
}

func TestGetTransactionsResponse_JSON(t *testing.T) {
	resp := GetTransactionsResponse{}
	resp.Data.Transactions = []TransactionDetail{
		{ID: "txn-1", PayeeName: "Shop", CategoryID: "cat-1"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GetTransactionsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Data.Transactions) != 1 {
		t.Fatalf("Expected 1 transaction, got %d", len(decoded.Data.Transactions))
	}
	if decoded.Data.Transactions[0].PayeeName != "Shop" {
		t.Errorf("PayeeName = %q, want %q", decoded.Data.Transactions[0].PayeeName, "Shop")
	}
}

func TestCreateTransactionsResponse_DuplicateImportIDs_JSON(t *testing.T) {
	raw := `{
		"data": {
			"transaction_ids": ["txn-1"],
			"duplicate_import_ids": ["YNAB:abc123", "YNAB:def456"]
		}
	}`

	var resp CreateTransactionsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(resp.Data.DuplicateImportIDs) != 2 {
		t.Fatalf("DuplicateImportIDs len = %d, want 2", len(resp.Data.DuplicateImportIDs))
	}
	if resp.Data.DuplicateImportIDs[0] != "YNAB:abc123" {
		t.Errorf("DuplicateImportIDs[0] = %q, want %q", resp.Data.DuplicateImportIDs[0], "YNAB:abc123")
	}
	if resp.Data.DuplicateImportIDs[1] != "YNAB:def456" {
		t.Errorf("DuplicateImportIDs[1] = %q, want %q", resp.Data.DuplicateImportIDs[1], "YNAB:def456")
	}
}

func TestCreateTransactionsResponse_DuplicateImportIDs_OmittedWhenEmpty(t *testing.T) {
	raw := `{"data": {"transaction_ids": ["txn-1"]}}`

	var resp CreateTransactionsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(resp.Data.DuplicateImportIDs) != 0 {
		t.Errorf("DuplicateImportIDs should be empty when absent, got %v", resp.Data.DuplicateImportIDs)
	}
}

func TestGetCategoriesResponse_JSON(t *testing.T) {
	resp := GetCategoriesResponse{}
	resp.Data.CategoryGroups = []CategoryGroup{
		{
			ID:   "group-1",
			Name: "Food",
			Categories: []CategoryItem{
				{ID: "cat-1", Name: "Groceries", Deleted: false, Hidden: false},
				{ID: "cat-2", Name: "Restaurants", Deleted: true, Hidden: false},
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GetCategoriesResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Data.CategoryGroups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(decoded.Data.CategoryGroups))
	}
	group := decoded.Data.CategoryGroups[0]
	if group.Name != "Food" {
		t.Errorf("Group Name = %q, want %q", group.Name, "Food")
	}
	if len(group.Categories) != 2 {
		t.Fatalf("Expected 2 categories, got %d", len(group.Categories))
	}
	if !group.Categories[1].Deleted {
		t.Error("Expected second category to be deleted")
	}
}
