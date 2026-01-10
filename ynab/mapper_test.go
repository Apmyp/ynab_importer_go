package ynab

import (
	"testing"
	"time"

	"github.com/apmyp/ynab_importer_go/bagoup"
	"github.com/apmyp/ynab_importer_go/template"
)

func TestNewMapper(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
		{YNABAccountID: "account-2", Last4: "5678"},
	}

	mapper := NewMapper(accounts)
	if mapper == nil {
		t.Error("NewMapper() returned nil")
	}
}

func TestMapper_MatchAccount(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
		{YNABAccountID: "account-2", Last4: "5678"},
	}
	mapper := NewMapper(accounts)

	tests := []struct {
		name    string
		card    string
		want    string
		wantErr bool
	}{
		{
			name:    "match with standard format",
			card:    "9..1234",
			want:    "account-1",
			wantErr: false,
		},
		{
			name:    "match second account",
			card:    "7..5678",
			want:    "account-2",
			wantErr: false,
		},
		{
			name:    "match with different mask",
			card:    "*1234",
			want:    "account-1",
			wantErr: false,
		},
		{
			name:    "no match",
			card:    "9..9999",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty card",
			card:    "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &template.Transaction{Card: tt.card}
			got, err := mapper.MatchAccount(tx)
			if (err != nil) != tt.wantErr {
				t.Errorf("MatchAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MatchAccount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapper_GenerateImportID(t *testing.T) {
	mapper := NewMapper([]YNABAccount{})

	msg := &bagoup.Message{
		Timestamp: time.Date(2026, 1, 10, 15, 30, 45, 0, time.UTC),
		Sender:    "102",
		Content:   "Test transaction",
	}

	tx := &template.Transaction{
		Card: "9..1234",
		Original: template.Amount{
			Value:    100.50,
			Currency: "MDL",
		},
		Address: "Test Merchant",
	}

	// Generate import ID twice - should be same (deterministic)
	id1 := mapper.GenerateImportID(msg, tx)
	id2 := mapper.GenerateImportID(msg, tx)

	if id1 != id2 {
		t.Error("GenerateImportID() should be deterministic")
	}

	if id1 == "" {
		t.Error("GenerateImportID() returned empty string")
	}

	// Should start with YNAB: prefix
	if len(id1) < 5 || id1[:5] != "YNAB:" {
		t.Errorf("GenerateImportID() = %v, should start with 'YNAB:'", id1)
	}

	// Different transactions should generate different IDs
	tx2 := &template.Transaction{
		Card: "9..5678",
		Original: template.Amount{
			Value:    200.00,
			Currency: "MDL",
		},
		Address: "Another Merchant",
	}

	id3 := mapper.GenerateImportID(msg, tx2)
	if id1 == id3 {
		t.Error("Different transactions should generate different import IDs")
	}
}

func TestMapper_MapTransaction(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
	}
	mapper := NewMapper(accounts)

	msg := &bagoup.Message{
		Timestamp: time.Date(2026, 1, 10, 15, 30, 45, 0, time.UTC),
		Sender:    "102",
	}

	tx := &template.Transaction{
		Operation: "Debitare",
		Card:      "9..1234",
		Status:    "Odobrena",
		Original: template.Amount{
			Value:    100.50,
			Currency: "MDL",
		},
		Converted: template.Amount{
			Value:    100.50,
			Currency: "MDL",
		},
		Address: "Test Merchant",
	}

	payload, err := mapper.MapTransaction(msg, tx)
	if err != nil {
		t.Fatalf("MapTransaction() error = %v", err)
	}

	if payload.AccountID != "account-1" {
		t.Errorf("AccountID = %v, want account-1", payload.AccountID)
	}

	if payload.Date != "2026-01-10" {
		t.Errorf("Date = %v, want 2026-01-10", payload.Date)
	}

	// Debitare should be negative
	if payload.Amount != -100500 {
		t.Errorf("Amount = %v, want -100500 (100.50 MDL in milliunits)", payload.Amount)
	}

	if payload.PayeeName != "Test Merchant" {
		t.Errorf("PayeeName = %v, want Test Merchant", payload.PayeeName)
	}

	if payload.Cleared != "cleared" {
		t.Errorf("Cleared = %v, want cleared", payload.Cleared)
	}

	if payload.ImportID == "" {
		t.Error("ImportID should not be empty")
	}
}

func TestMapper_MapTransaction_Suplinire(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
	}
	mapper := NewMapper(accounts)

	msg := &bagoup.Message{
		Timestamp: time.Date(2026, 1, 10, 15, 30, 45, 0, time.UTC),
	}

	tx := &template.Transaction{
		Operation: "Suplinire",
		Card:      "9..1234",
		Converted: template.Amount{
			Value:    1000.00,
			Currency: "MDL",
		},
		Address: "Salary",
	}

	payload, err := mapper.MapTransaction(msg, tx)
	if err != nil {
		t.Fatalf("MapTransaction() error = %v", err)
	}

	// Suplinire should be positive
	if payload.Amount != 1000000 {
		t.Errorf("Amount = %v, want 1000000 (1000.00 MDL in milliunits)", payload.Amount)
	}
}

func TestMapper_MapTransaction_NoAccountMatch(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "9999"},
	}
	mapper := NewMapper(accounts)

	msg := &bagoup.Message{
		Timestamp: time.Date(2026, 1, 10, 15, 30, 45, 0, time.UTC),
	}

	tx := &template.Transaction{
		Card: "9..1234",
		Converted: template.Amount{
			Value:    100.00,
			Currency: "MDL",
		},
	}

	_, err := mapper.MapTransaction(msg, tx)
	if err == nil {
		t.Error("MapTransaction() should return error when no account matches")
	}
}
