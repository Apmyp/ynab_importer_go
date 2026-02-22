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

func (m *mockClient) GetTransactions(budgetID string, sinceDate string) (*GetTransactionsResponse, error) {
	return &GetTransactionsResponse{}, nil
}

func (m *mockClient) GetCategories(budgetID string) (*GetCategoriesResponse, error) {
	return &GetCategoriesResponse{}, nil
}

func (m *mockClient) DeleteTransaction(budgetID, transactionID string) error {
	return nil
}

func TestNewSyncer(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	client := &mockClient{}
	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")

	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)
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
					TransactionIDs     []string `json:"transaction_ids"`
					DuplicateImportIDs []string `json:"duplicate_import_ids,omitempty"`
					Transactions       []struct {
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

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)

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

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
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

	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)

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

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)

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
					TransactionIDs     []string `json:"transaction_ids"`
					DuplicateImportIDs []string `json:"duplicate_import_ids,omitempty"`
					Transactions       []struct {
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

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)

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

func TestTransactionPayload_HasTransferAccountID(t *testing.T) {
	payload := TransactionPayload{
		AccountID:         "acc-1",
		TransferAccountID: "acc-2",
	}
	if payload.TransferAccountID != "acc-2" {
		t.Errorf("TransferAccountID = %q, want %q", payload.TransferAccountID, "acc-2")
	}
}

func TestDetectTransferPairs_DetectsPair(t *testing.T) {
	baseTime := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)

	messages := []*message.Message{
		{Timestamp: baseTime},
		{Timestamp: baseTime.Add(2 * time.Minute)},
	}
	transactions := []*template.Transaction{
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..2222", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Suplinire"},
	}

	pairs := detectTransferPairs(messages, transactions)

	if len(pairs) != 1 {
		t.Fatalf("detectTransferPairs() returned %d pairs, want 1", len(pairs))
	}
	if pairs[0] != 1 {
		t.Errorf("pairs[0] = %d, want 1 (credit index)", pairs[0])
	}
}

func TestDetectTransferPairs_NoPairWhenSameAccount(t *testing.T) {
	baseTime := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)

	messages := []*message.Message{
		{Timestamp: baseTime},
		{Timestamp: baseTime.Add(2 * time.Minute)},
	}
	transactions := []*template.Transaction{
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Suplinire"},
	}

	pairs := detectTransferPairs(messages, transactions)

	if len(pairs) != 0 {
		t.Errorf("detectTransferPairs() returned %d pairs, want 0 (same account)", len(pairs))
	}
}

func TestDetectTransferPairs_NoPairWhenOutsideTimeWindow(t *testing.T) {
	baseTime := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)

	messages := []*message.Message{
		{Timestamp: baseTime},
		{Timestamp: baseTime.Add(10 * time.Minute)},
	}
	transactions := []*template.Transaction{
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..2222", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Suplinire"},
	}

	pairs := detectTransferPairs(messages, transactions)

	if len(pairs) != 0 {
		t.Errorf("detectTransferPairs() returned %d pairs, want 0 (outside time window)", len(pairs))
	}
}

func TestDetectTransferPairs_NoPairWhenAmountDiffers(t *testing.T) {
	baseTime := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)

	messages := []*message.Message{
		{Timestamp: baseTime},
		{Timestamp: baseTime.Add(1 * time.Minute)},
	}
	transactions := []*template.Transaction{
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..2222", Converted: template.Amount{Value: 450.00, Currency: "MDL"}, Operation: "Suplinire"},
	}

	pairs := detectTransferPairs(messages, transactions)

	if len(pairs) != 0 {
		t.Errorf("detectTransferPairs() returned %d pairs, want 0 (different amounts)", len(pairs))
	}
}

func TestSyncer_Sync_TransferPair_SkipsCreditSide(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	var capturedTransactions []TransactionPayload
	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			capturedTransactions = transactions
			response := &CreateTransactionsResponse{}
			for range transactions {
				response.Data.TransactionIDs = append(response.Data.TransactionIDs, "txn-1")
			}
			return response, nil
		},
	}

	mapper := NewMapper([]YNABAccount{
		{YNABAccountID: "acc-debit", Last4: "1111"},
		{YNABAccountID: "acc-credit", Last4: "2222"},
	}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)

	baseTime := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	messages := []*message.Message{
		{Timestamp: baseTime, Sender: "102"},
		{Timestamp: baseTime.Add(2 * time.Minute), Sender: "102"},
	}
	transactions := []*template.Transaction{
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare", Address: "Transfer"},
		{Card: "9..2222", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Suplinire", Address: "Transfer"},
	}

	result, err := syncer.Sync(messages, transactions)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if result.Synced != 1 {
		t.Errorf("Synced = %d, want 1 (only debit side)", result.Synced)
	}

	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (credit side skipped)", result.Skipped)
	}

	if len(capturedTransactions) != 1 {
		t.Fatalf("Expected 1 transaction sent to API, got %d", len(capturedTransactions))
	}

	sent := capturedTransactions[0]
	if sent.AccountID != "acc-debit" {
		t.Errorf("AccountID = %q, want %q", sent.AccountID, "acc-debit")
	}
	if sent.TransferAccountID != "acc-credit" {
		t.Errorf("TransferAccountID = %q, want %q", sent.TransferAccountID, "acc-credit")
	}
	if sent.Amount >= 0 {
		t.Errorf("Amount = %d, want negative (debit)", sent.Amount)
	}
}

func TestSyncer_Sync_NonTransferTransactionsUnaffected(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	var capturedTransactions []TransactionPayload
	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			capturedTransactions = transactions
			response := &CreateTransactionsResponse{}
			for range transactions {
				response.Data.TransactionIDs = append(response.Data.TransactionIDs, "txn-1")
			}
			return response, nil
		},
	}

	mapper := NewMapper([]YNABAccount{
		{YNABAccountID: "acc-1", Last4: "1111"},
	}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)

	baseTime := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	messages := []*message.Message{
		{Timestamp: baseTime, Sender: "102"},
	}
	transactions := []*template.Transaction{
		{Card: "9..1111", Converted: template.Amount{Value: 100.00, Currency: "MDL"}, Operation: "Debitare", Address: "Coffee Shop"},
	}

	result, err := syncer.Sync(messages, transactions)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if result.Synced != 1 {
		t.Errorf("Synced = %d, want 1", result.Synced)
	}

	if len(capturedTransactions) != 1 {
		t.Fatalf("Expected 1 transaction, got %d", len(capturedTransactions))
	}

	if capturedTransactions[0].TransferAccountID != "" {
		t.Errorf("TransferAccountID = %q, want empty for non-transfer", capturedTransactions[0].TransferAccountID)
	}
}

func TestDetectTransferPairs_CreditUsedOnlyOnce(t *testing.T) {
	baseTime := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)

	messages := []*message.Message{
		{Timestamp: baseTime},
		{Timestamp: baseTime.Add(1 * time.Minute)},
		{Timestamp: baseTime.Add(2 * time.Minute)},
	}
	transactions := []*template.Transaction{
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..3333", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..2222", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Suplinire"},
	}

	pairs := detectTransferPairs(messages, transactions)

	if len(pairs) != 1 {
		t.Fatalf("detectTransferPairs() returned %d pairs, want 1 (credit can only be used once)", len(pairs))
	}
	if _, hasFirst := pairs[0]; !hasFirst {
		t.Error("Expected first debit (index 0) to be paired, not second")
	}
}

func TestSyncer_Sync_TransferPair_CreditAccountNotFound(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			response := &CreateTransactionsResponse{}
			for range transactions {
				response.Data.TransactionIDs = append(response.Data.TransactionIDs, "txn-1")
			}
			return response, nil
		},
	}

	// Mapper only knows about the debit account, not the credit account
	mapper := NewMapper([]YNABAccount{
		{YNABAccountID: "acc-debit", Last4: "1111"},
	}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)

	baseTime := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	messages := []*message.Message{
		{Timestamp: baseTime, Sender: "102"},
		{Timestamp: baseTime.Add(2 * time.Minute), Sender: "102"},
	}
	transactions := []*template.Transaction{
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare", Address: "Transfer"},
		{Card: "9..2222", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Suplinire", Address: "Transfer"},
	}

	result, err := syncer.Sync(messages, transactions)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if result.Synced != 0 {
		t.Errorf("Synced = %d, want 0 (credit account not found, debit skipped)", result.Synced)
	}

	if len(result.Failed) == 0 {
		t.Error("Expected failure recorded for unmapped credit account")
	}
}

func TestSyncResult_HasUnknownPayees(t *testing.T) {
	result := &SyncResult{
		Total:         1,
		Synced:        1,
		UnknownPayees: []string{"Coffee Shop"},
	}
	if len(result.UnknownPayees) != 1 {
		t.Errorf("UnknownPayees = %v, want 1 entry", result.UnknownPayees)
	}
}

func TestSyncer_Sync_TracksUnknownPayees(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			resp := &CreateTransactionsResponse{}
			for range transactions {
				resp.Data.TransactionIDs = append(resp.Data.TransactionIDs, "txn-1")
			}
			return resp, nil
		},
	}

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)

	messages := []*message.Message{
		{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)},
	}
	transactions := []*template.Transaction{
		{Card: "9..1234", Converted: template.Amount{Value: 10, Currency: "MDL"}, Operation: "Debitare", Address: "Coffee Shop"},
	}

	result, err := syncer.Sync(messages, transactions)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if len(result.UnknownPayees) != 1 {
		t.Errorf("UnknownPayees = %v, want 1 entry", result.UnknownPayees)
	}
	if result.UnknownPayees[0] != "Coffee Shop" {
		t.Errorf("UnknownPayees[0] = %q, want Coffee Shop", result.UnknownPayees[0])
	}
}

func TestSyncer_Sync_NoUnknownPayees_WhenCategoryAssigned(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store, _ := NewSyncStore(filePath)
	defer store.Close()

	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			resp := &CreateTransactionsResponse{}
			for range transactions {
				resp.Data.TransactionIDs = append(resp.Data.TransactionIDs, "txn-1")
			}
			return resp, nil
		},
	}

	categoryStore := NewCategoryStore(filePath)
	_ = categoryStore.Set("Coffee Shop", "cat-food")
	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, categoryStore)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)

	messages := []*message.Message{
		{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)},
	}
	transactions := []*template.Transaction{
		{Card: "9..1234", Converted: template.Amount{Value: 10, Currency: "MDL"}, Operation: "Debitare", Address: "Coffee Shop"},
	}

	result, err := syncer.Sync(messages, transactions)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if len(result.UnknownPayees) != 0 {
		t.Errorf("UnknownPayees = %v, want empty (category was assigned)", result.UnknownPayees)
	}
}

func TestSyncer_Sync_WarnsDuplicateImportIDs(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			resp := &CreateTransactionsResponse{}
			resp.Data.DuplicateImportIDs = []string{"YNAB:abc123", "YNAB:def456"}
			return resp, nil
		},
	}

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)

	messages := []*message.Message{
		{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)},
	}
	transactions := []*template.Transaction{
		{Card: "9..1234", Converted: template.Amount{Value: 10, Currency: "MDL"}, Operation: "Debitare"},
	}

	result, err := syncer.Sync(messages, transactions)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if len(result.Warnings) != 2 {
		t.Fatalf("Warnings = %v, want 2 entries", result.Warnings)
	}
	if result.Warnings[0] != "Warning: YNAB skipped as duplicate: YNAB:abc123" {
		t.Errorf("Warnings[0] = %q, want warning about YNAB:abc123", result.Warnings[0])
	}
	if result.Warnings[1] != "Warning: YNAB skipped as duplicate: YNAB:def456" {
		t.Errorf("Warnings[1] = %q, want warning about YNAB:def456", result.Warnings[1])
	}
}

func TestSyncer_Sync_ReimportMode_ClearsImportIDInPayload(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	var capturedPayloads []TransactionPayload
	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			capturedPayloads = append(capturedPayloads, transactions...)
			resp := &CreateTransactionsResponse{}
			for range transactions {
				resp.Data.TransactionIDs = append(resp.Data.TransactionIDs, "txn-1")
			}
			return resp, nil
		},
	}

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, true)

	msg := &message.Message{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)}
	tx := &template.Transaction{
		Card:      "9..1234",
		Converted: template.Amount{Value: 100, Currency: "MDL"},
		Operation: "Debitare",
	}

	result, err := syncer.Sync([]*message.Message{msg}, []*template.Transaction{tx})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if result.Synced != 1 {
		t.Fatalf("Synced = %d, want 1", result.Synced)
	}

	if len(capturedPayloads) != 1 {
		t.Fatalf("Expected 1 payload sent to API, got %d", len(capturedPayloads))
	}

	if capturedPayloads[0].ImportID != "" {
		t.Errorf("ImportID = %q, want empty in reimport mode", capturedPayloads[0].ImportID)
	}

	computedImportID := mapper.GenerateImportID(msg, tx)
	synced, err := store.IsSynced(computedImportID)
	if err != nil {
		t.Fatalf("IsSynced() error = %v", err)
	}
	if !synced {
		t.Errorf("local store should record computed importID %q after reimport sync", computedImportID)
	}
}

func TestSyncer_Sync_NormalMode_KeepsImportIDInPayload(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	var capturedPayloads []TransactionPayload
	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			capturedPayloads = append(capturedPayloads, transactions...)
			resp := &CreateTransactionsResponse{}
			for range transactions {
				resp.Data.TransactionIDs = append(resp.Data.TransactionIDs, "txn-1")
			}
			return resp, nil
		},
	}

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)

	msg := &message.Message{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)}
	tx := &template.Transaction{
		Card:      "9..1234",
		Converted: template.Amount{Value: 100, Currency: "MDL"},
		Operation: "Debitare",
	}

	result, err := syncer.Sync([]*message.Message{msg}, []*template.Transaction{tx})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if result.Synced != 1 {
		t.Fatalf("Synced = %d, want 1", result.Synced)
	}

	if len(capturedPayloads) != 1 {
		t.Fatalf("Expected 1 payload sent to API, got %d", len(capturedPayloads))
	}

	if capturedPayloads[0].ImportID == "" {
		t.Errorf("ImportID should not be empty in normal mode")
	}
}

func TestSyncer_Sync_RecordsSyncWithTransactionDate(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewSyncStore(dir + "/data.json")
	defer store.Close()

	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			resp := &CreateTransactionsResponse{}
			for range transactions {
				resp.Data.TransactionIDs = append(resp.Data.TransactionIDs, "txn-1")
			}
			return resp, nil
		},
	}

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)

	txDate := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	messages := []*message.Message{
		{Timestamp: txDate, Sender: "102"},
	}
	transactions := []*template.Transaction{
		{Card: "9..1234", Converted: template.Amount{Value: 100, Currency: "MDL"}, Operation: "Debitare"},
	}

	_, err := syncer.Sync(messages, transactions)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	records, err := store.GetAllSynced()
	if err != nil {
		t.Fatalf("GetAllSynced() error = %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("GetAllSynced() returned %d records, want 1", len(records))
	}

	if records[0].TransactionDate != "2026-01-15" {
		t.Errorf("TransactionDate = %q, want %q", records[0].TransactionDate, "2026-01-15")
	}
}

func TestSyncer_Sync_UnknownPayees_Deduplicated(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			resp := &CreateTransactionsResponse{}
			for range transactions {
				resp.Data.TransactionIDs = append(resp.Data.TransactionIDs, "txn-1")
			}
			return resp, nil
		},
	}

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false)

	messages := []*message.Message{
		{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)},
		{Timestamp: time.Date(2026, 1, 11, 10, 0, 0, 0, time.UTC)},
	}
	transactions := []*template.Transaction{
		{Card: "9..1234", Converted: template.Amount{Value: 10, Currency: "MDL"}, Operation: "Debitare", Address: "Coffee Shop"},
		{Card: "9..1234", Converted: template.Amount{Value: 20, Currency: "MDL"}, Operation: "Debitare", Address: "Coffee Shop"},
	}

	result, err := syncer.Sync(messages, transactions)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if len(result.UnknownPayees) != 1 {
		t.Errorf("UnknownPayees = %v, want 1 (deduplicated)", result.UnknownPayees)
	}
}
