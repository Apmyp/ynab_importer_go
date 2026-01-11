package ynab

import (
	"errors"
	"testing"
	"time"

	"github.com/apmyp/ynab_importer_go/message"
	"github.com/apmyp/ynab_importer_go/template"
)

// Mock client for testing
type mockClient struct {
	createTransactionsFunc func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error)
	getAccountsFunc        func(budgetID string) (*GetAccountsResponse, error)
	createAccountFunc      func(budgetID string, payload CreateAccountPayload) (*CreateAccountResponse, error)
}

func (m *mockClient) CreateTransactions(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
	if m.createTransactionsFunc != nil {
		return m.createTransactionsFunc(budgetID, transactions)
	}
	return &CreateTransactionsResponse{}, nil
}

func (m *mockClient) GetAccounts(budgetID string) (*GetAccountsResponse, error) {
	if m.getAccountsFunc != nil {
		return m.getAccountsFunc(budgetID)
	}
	return &GetAccountsResponse{}, nil
}

func (m *mockClient) CreateAccount(budgetID string, payload CreateAccountPayload) (*CreateAccountResponse, error) {
	if m.createAccountFunc != nil {
		return m.createAccountFunc(budgetID, payload)
	}
	return &CreateAccountResponse{}, nil
}

func TestNewSyncer(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	client := &mockClient{}
	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}})
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")

	syncer := NewSyncer(store, client, mapper, "test-budget", startDate)
	if syncer == nil {
		t.Error("NewSyncer() returned nil")
	}
}

func TestSyncer_Sync_FiltersByDate(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	var capturedTransactions []TransactionPayload
	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			capturedTransactions = transactions
			response := &CreateTransactionsResponse{
				Data: struct {
					TransactionIDs []string `json:"transaction_ids"`
					Transactions   []struct {
						ID       string `json:"id"`
						ImportID string `json:"import_id"`
					} `json:"transactions,omitempty"`
				}{},
			}
			for range transactions {
				response.Data.TransactionIDs = append(response.Data.TransactionIDs, "txn-1")
			}
			return response, nil
		},
	}

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}})
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate)

	// Create messages - one before startDate, one after
	messages := []*message.Message{
		{Timestamp: time.Date(2025, 12, 31, 10, 0, 0, 0, time.UTC), Sender: "102"},
		{Timestamp: time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC), Sender: "102"},
		{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC), Sender: "102"},
	}

	transactions := []*template.Transaction{
		{Card: "9..1234", Converted: template.Amount{Value: 100, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..1234", Converted: template.Amount{Value: 200, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..1234", Converted: template.Amount{Value: 300, Currency: "MDL"}, Operation: "Debitare"},
	}

	result, err := syncer.Sync(messages, transactions)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	// Should only sync 2 transactions (after start date)
	if result.Synced != 2 {
		t.Errorf("Synced = %d, want 2", result.Synced)
	}

	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (before start date)", result.Skipped)
	}

	if len(capturedTransactions) != 2 {
		t.Errorf("Expected 2 transactions sent to API, got %d", len(capturedTransactions))
	}
}

func TestSyncer_Sync_SkipsAlreadySynced(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewSyncStore(dir + "/data.json")
	defer store.Close()

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}})
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")

	msg := &message.Message{
		Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
		Sender:    "102",
	}
	tx := &template.Transaction{
		Card:      "9..1234",
		Converted: template.Amount{Value: 100, Currency: "MDL"},
		Operation: "Debitare",
	}

	// Record as already synced
	importID := mapper.GenerateImportID(msg, tx)
	store.RecordSync(&SyncRecord{ImportID: importID, SyncedAt: time.Now()})

	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			if len(transactions) > 0 {
				t.Error("Should not send any transactions (already synced)")
			}
			return &CreateTransactionsResponse{}, nil
		},
	}

	syncer := NewSyncer(store, client, mapper, "test-budget", startDate)

	result, err := syncer.Sync([]*message.Message{msg}, []*template.Transaction{tx})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if result.Synced != 0 {
		t.Errorf("Synced = %d, want 0 (already synced)", result.Synced)
	}

	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (already synced)", result.Skipped)
	}
}

func TestSyncer_Sync_HandlesAPIError(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			return nil, errors.New("API error")
		},
	}

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}})
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate)

	msg := &message.Message{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)}
	tx := &template.Transaction{
		Card:      "9..1234",
		Converted: template.Amount{Value: 100, Currency: "MDL"},
		Operation: "Debitare",
	}

	_, err := syncer.Sync([]*message.Message{msg}, []*template.Transaction{tx})
	if err == nil {
		t.Error("Sync() should return error when API call fails")
	}
}

func TestSyncer_Sync_BatchesTransactions(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	batchCount := 0
	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			batchCount++
			if len(transactions) > 100 {
				t.Errorf("Batch size = %d, should be <= 100", len(transactions))
			}
			response := &CreateTransactionsResponse{
				Data: struct {
					TransactionIDs []string `json:"transaction_ids"`
					Transactions   []struct {
						ID       string `json:"id"`
						ImportID string `json:"import_id"`
					} `json:"transactions,omitempty"`
				}{},
			}
			for range transactions {
				response.Data.TransactionIDs = append(response.Data.TransactionIDs, "txn-1")
			}
			return response, nil
		},
	}

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}})
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate)

	// Create 150 transactions
	var messages []*message.Message
	var transactions []*template.Transaction
	for i := 0; i < 150; i++ {
		messages = append(messages, &message.Message{
			Timestamp: time.Date(2026, 1, 10, 10, 0, i, 0, time.UTC),
			Sender:    "102",
		})
		transactions = append(transactions, &template.Transaction{
			Card:      "9..1234",
			Converted: template.Amount{Value: float64(100 + i), Currency: "MDL"},
			Operation: "Debitare",
		})
	}

	result, err := syncer.Sync(messages, transactions)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	// Should create 2 batches (100 + 50)
	if batchCount != 2 {
		t.Errorf("batchCount = %d, want 2 (100 + 50)", batchCount)
	}

	if result.Synced != 150 {
		t.Errorf("Synced = %d, want 150", result.Synced)
	}
}
