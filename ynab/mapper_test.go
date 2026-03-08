package ynab

import (
	"testing"
	"time"

	"github.com/apmyp/ynab_importer_go/message"
	"github.com/apmyp/ynab_importer_go/template"
)

func TestNewMapper(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
		{YNABAccountID: "account-2", Last4: "5678"},
	}

	mapper := NewMapper(accounts, nil)
	if mapper == nil {
		t.Error("NewMapper() returned nil")
	}
}

func TestMapper_MatchAccount(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
		{YNABAccountID: "account-2", Last4: "5678"},
		{YNABAccountID: "account-6345", Last4: "6345"},
	}
	mapper := NewMapper(accounts, nil)

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
			name:    "match card 6345",
			card:    "9..6345",
			want:    "account-6345",
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
	mapper := NewMapper([]YNABAccount{}, nil)

	msg := &message.Message{
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
	mapper := NewMapper(accounts, nil)

	msg := &message.Message{
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

	if payload.PayeeName != "" {
		t.Errorf("PayeeName = %v, want empty (Debitare uses memo)", payload.PayeeName)
	}
	if payload.Memo != "Test Merchant" {
		t.Errorf("Memo = %v, want Test Merchant (address as memo for Debitare)", payload.Memo)
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
	mapper := NewMapper(accounts, nil)

	msg := &message.Message{
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
	mapper := NewMapper(accounts, nil)

	msg := &message.Message{
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

func TestBuildMemo_StandardTransaction(t *testing.T) {
	// Standard operation + standard status = empty memo
	tx := &template.Transaction{
		Operation: "Tovary i uslugi",
		Status:    "Odobrena",
	}

	memo := buildMemo(tx)
	if memo != "" {
		t.Errorf("buildMemo() = %q, want empty string for standard transaction", memo)
	}
}

func TestBuildMemo_StandardOperationEmptyStatus(t *testing.T) {
	// Standard operation + empty status = empty memo
	tx := &template.Transaction{
		Operation: "Debitare",
		Status:    "",
	}

	memo := buildMemo(tx)
	if memo != "" {
		t.Errorf("buildMemo() = %q, want empty string", memo)
	}
}

func TestBuildMemo_NonStandardStatus(t *testing.T) {
	// Standard operation + non-standard status = status in memo
	tx := &template.Transaction{
		Operation: "Tovary i uslugi",
		Status:    "Pending Review",
	}

	memo := buildMemo(tx)
	if memo != "Pending Review" {
		t.Errorf("buildMemo() = %q, want 'Pending Review'", memo)
	}
}

func TestBuildMemo_NonStandardOperation(t *testing.T) {
	// Non-standard operation + standard status = operation in memo
	tx := &template.Transaction{
		Operation: "Special Payment",
		Status:    "Odobrena",
	}

	memo := buildMemo(tx)
	if memo != "Special Payment" {
		t.Errorf("buildMemo() = %q, want 'Special Payment'", memo)
	}
}

func TestBuildMemo_BothNonStandard(t *testing.T) {
	// Both non-standard = both in memo
	tx := &template.Transaction{
		Operation: "Special Payment",
		Status:    "Requires Approval",
	}

	memo := buildMemo(tx)
	if memo != "Special Payment - Requires Approval" {
		t.Errorf("buildMemo() = %q, want 'Special Payment - Requires Approval'", memo)
	}
}

func TestMapper_MapTransaction_StandardMemoIsEmpty(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
	}
	mapper := NewMapper(accounts, nil)

	msg := &message.Message{
		Timestamp: time.Date(2026, 1, 10, 15, 30, 45, 0, time.UTC),
	}

	tx := &template.Transaction{
		Operation: "Tovary i uslugi",
		Status:    "Odobrena",
		Card:      "9..1234",
		Converted: template.Amount{
			Value:    100.00,
			Currency: "MDL",
		},
		Address: "Test Shop",
	}

	payload, err := mapper.MapTransaction(msg, tx)
	if err != nil {
		t.Fatalf("MapTransaction() error = %v", err)
	}

	if payload.Memo != "" {
		t.Errorf("Memo = %q, want empty string for standard transaction", payload.Memo)
	}
}

func TestMapper_MapTransaction_RaznyeVyplaty_IsDebit(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1972"},
	}
	mapper := NewMapper(accounts, nil)

	msg := &message.Message{
		Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
	}

	tx := &template.Transaction{
		Operation: "Raznye vyplaty *4335",
		Card:      "*1972",
		Converted: template.Amount{
			Value:    50000.00,
			Currency: "MDL",
		},
		Address: "Transfer",
	}

	payload, err := mapper.MapTransaction(msg, tx)
	if err != nil {
		t.Fatalf("MapTransaction() error = %v", err)
	}

	if payload.Amount != -50000000 {
		t.Errorf("Amount = %v, want -50000000 (debit/outflow)", payload.Amount)
	}
}

func TestMapper_MapTransaction_NalogNaDoxodyPoVkladu_IsDebit(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
	}
	mapper := NewMapper(accounts, nil)

	msg := &message.Message{
		Timestamp: time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC),
	}

	tx := &template.Transaction{
		Operation: "Nalog na doxody po vkladu",
		Card:      "9..1234",
		Converted: template.Amount{
			Value:    50.00,
			Currency: "MDL",
		},
		Address: "Tax Office",
	}

	payload, err := mapper.MapTransaction(msg, tx)
	if err != nil {
		t.Fatalf("MapTransaction() error = %v", err)
	}

	if payload.Amount != -50000 {
		t.Errorf("Amount = %v, want -50000 (debit/outflow)", payload.Amount)
	}
}

func TestMapper_MapTransaction_Debitare_AddressAsMemo(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "6345"},
	}
	mapper := NewMapper(accounts, nil)

	msg := &message.Message{
		Timestamp: time.Date(2026, 2, 26, 12, 8, 22, 0, time.UTC),
	}

	tx := &template.Transaction{
		Operation: "Debitare",
		Card:      "4..6345",
		Converted: template.Amount{Value: 150000.00, Currency: "MDL"},
		Address:   "Plata OP /  0 - MAIB : transfer pe card meu",
	}

	payload, err := mapper.MapTransaction(msg, tx)
	if err != nil {
		t.Fatalf("MapTransaction() error = %v", err)
	}

	if payload.PayeeName != "" {
		t.Errorf("PayeeName = %q, want empty for Debitare transaction", payload.PayeeName)
	}
	if payload.Memo != "Plata OP /  0 - MAIB : transfer pe card meu" {
		t.Errorf("Memo = %q, want address as memo for Debitare transaction", payload.Memo)
	}
}

func TestMapper_MapTransaction_Suplinire_AddressAsMemo(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "6345"},
	}
	mapper := NewMapper(accounts, nil)

	msg := &message.Message{
		Timestamp: time.Date(2026, 3, 5, 9, 35, 0, 0, time.UTC),
	}

	tx := &template.Transaction{
		Operation: "Suplinire",
		Card:      "4..6345",
		Converted: template.Amount{Value: 58211.54, Currency: "MDL"},
		Address:   "Plata primei anuale in baza conventiei de salarizare",
	}

	payload, err := mapper.MapTransaction(msg, tx)
	if err != nil {
		t.Fatalf("MapTransaction() error = %v", err)
	}

	if payload.PayeeName != "" {
		t.Errorf("PayeeName = %q, want empty for Suplinire transaction", payload.PayeeName)
	}
	if payload.Memo != "Plata primei anuale in baza conventiei de salarizare" {
		t.Errorf("Memo = %q, want address as memo for Suplinire transaction", payload.Memo)
	}
}

func TestMapper_MapTransaction_PlataSalarialaLuna_NoPayeeAddressAsMemo(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
	}
	mapper := NewMapper(accounts, nil)

	msg := &message.Message{
		Timestamp: time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC),
	}

	tx := &template.Transaction{
		Operation: "Suplinire",
		Card:      "9..1234",
		Converted: template.Amount{
			Value:    5000.00,
			Currency: "MDL",
		},
		Address: "Plata salariala luna ianuarie 2026",
	}

	payload, err := mapper.MapTransaction(msg, tx)
	if err != nil {
		t.Fatalf("MapTransaction() error = %v", err)
	}

	if payload.PayeeName != "" {
		t.Errorf("PayeeName = %q, want empty string for salary transaction", payload.PayeeName)
	}

	if payload.Memo != "Plata salariala luna ianuarie 2026" {
		t.Errorf("Memo = %q, want address as memo for salary transaction", payload.Memo)
	}
}

func TestMapper_MapTransaction_UspeshnoeReversirovanie_IsInflow(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
	}
	mapper := NewMapper(accounts, nil)

	msg := &message.Message{
		Timestamp: time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC),
	}

	tx := &template.Transaction{
		Operation: "Uspeshnoe reversirovanie",
		Card:      "9..1234",
		Converted: template.Amount{
			Value:    200.00,
			Currency: "MDL",
		},
		Address: "Some Merchant",
	}

	payload, err := mapper.MapTransaction(msg, tx)
	if err != nil {
		t.Fatalf("MapTransaction() error = %v", err)
	}

	if payload.Amount != 200000 {
		t.Errorf("Amount = %v, want 200000 (inflow/positive)", payload.Amount)
	}
}

func TestNewMapper_WithCategoryStore(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
	}
	filePath := t.TempDir() + "/data.json"
	categoryStore := NewCategoryStore(filePath)

	mapper := NewMapper(accounts, categoryStore)
	if mapper == nil {
		t.Error("NewMapper() with category store returned nil")
	}
}

func TestNewMapper_WithNilCategoryStore(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
	}
	mapper := NewMapper(accounts, nil)
	if mapper == nil {
		t.Error("NewMapper() with nil category store returned nil")
	}
}

func TestMapper_MapTransaction_AppliesCategory(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
	}
	filePath := t.TempDir() + "/data.json"
	categoryStore := NewCategoryStore(filePath)
	_ = categoryStore.Set("Coffee Shop", "cat-food-123")

	mapper := NewMapper(accounts, categoryStore)

	msg := &message.Message{
		Timestamp: time.Date(2026, 1, 10, 15, 30, 45, 0, time.UTC),
	}
	tx := &template.Transaction{
		Operation: "Debitare",
		Card:      "9..1234",
		Converted: template.Amount{Value: 10.00, Currency: "MDL"},
		Address:   "Coffee Shop",
	}

	payload, err := mapper.MapTransaction(msg, tx)
	if err != nil {
		t.Fatalf("MapTransaction() error = %v", err)
	}

	// Debitare puts Address in memo and clears payee, so no category is assigned.
	if payload.CategoryID != "" {
		t.Errorf("CategoryID = %q, want empty (no payee for Debitare)", payload.CategoryID)
	}
}

func TestMapper_MapTransaction_NoCategoryWhenUnknownPayee(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
	}
	filePath := t.TempDir() + "/data.json"
	categoryStore := NewCategoryStore(filePath)

	mapper := NewMapper(accounts, categoryStore)

	msg := &message.Message{
		Timestamp: time.Date(2026, 1, 10, 15, 30, 45, 0, time.UTC),
	}
	tx := &template.Transaction{
		Operation: "Debitare",
		Card:      "9..1234",
		Converted: template.Amount{Value: 10.00, Currency: "MDL"},
		Address:   "Unknown New Shop",
	}

	payload, err := mapper.MapTransaction(msg, tx)
	if err != nil {
		t.Fatalf("MapTransaction() error = %v", err)
	}

	if payload.CategoryID != "" {
		t.Errorf("CategoryID = %q, want empty for unknown payee", payload.CategoryID)
	}
}

func TestMapper_MapTransaction_NoCategoryWithNilStore(t *testing.T) {
	accounts := []YNABAccount{
		{YNABAccountID: "account-1", Last4: "1234"},
	}
	mapper := NewMapper(accounts, nil)

	msg := &message.Message{
		Timestamp: time.Date(2026, 1, 10, 15, 30, 45, 0, time.UTC),
	}
	tx := &template.Transaction{
		Operation: "Debitare",
		Card:      "9..1234",
		Converted: template.Amount{Value: 10.00, Currency: "MDL"},
		Address:   "Coffee Shop",
	}

	payload, err := mapper.MapTransaction(msg, tx)
	if err != nil {
		t.Fatalf("MapTransaction() error = %v", err)
	}

	if payload.CategoryID != "" {
		t.Errorf("CategoryID = %q, want empty when store is nil", payload.CategoryID)
	}
}
