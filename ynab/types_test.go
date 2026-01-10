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
