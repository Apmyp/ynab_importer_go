package ynab

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/apmyp/ynab_importer_go/analyzer"
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

// makeAnalyzed builds a slice of AnalyzedTransactions with no pairs for convenience.
func makeAnalyzed(msgs []*message.Message, txs []*template.Transaction) []analyzer.AnalyzedTransaction {
	result := make([]analyzer.AnalyzedTransaction, len(msgs))
	for i := range msgs {
		kind := analyzer.KindIncome
		if isDebit(txs[i].Operation) {
			kind = analyzer.KindPayment
		}
		result[i] = analyzer.AnalyzedTransaction{
			Message:     msgs[i],
			Transaction: txs[i],
			Kind:        kind,
			PairIndex:   -1,
		}
	}
	return result
}

func TestNewSyncer(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	client := &mockClient{}
	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")

	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 0)
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
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 0)

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

	result, err := syncer.Sync(makeAnalyzed(messages, transactions))
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

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

	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 0)

	result, err := syncer.Sync(makeAnalyzed([]*message.Message{msg}, []*template.Transaction{tx}))
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
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 0)

	analyzed := []analyzer.AnalyzedTransaction{
		{
			Message:     &message.Message{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)},
			Transaction: &template.Transaction{Card: "9..1234", Converted: template.Amount{Value: 100, Currency: "MDL"}, Operation: "Debitare"},
			Kind:        analyzer.KindPayment,
			PairIndex:   -1,
		},
	}

	_, err := syncer.Sync(analyzed)
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
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 0)

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

	result, err := syncer.Sync(makeAnalyzed(messages, transactions))
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

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
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 0)

	base := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	analyzed := []analyzer.AnalyzedTransaction{
		{
			Message:     &message.Message{Timestamp: base, Sender: "102"},
			Transaction: &template.Transaction{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
			Kind:        analyzer.KindTransfer,
			HasPair:     true,
			PairIndex:   1,
		},
		{
			Message:     &message.Message{Timestamp: base.Add(2 * time.Minute), Sender: "102"},
			Transaction: &template.Transaction{Card: "9..2222", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Suplinire"},
			Kind:        analyzer.KindCreditTransfer,
			HasPair:     true,
			PairIndex:   0,
		},
	}

	result, err := syncer.Sync(analyzed)
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

func TestSyncer_Sync_IncomeKind_SyncedAsPositive(t *testing.T) {
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
		{YNABAccountID: "acc-1", Last4: "1234"},
	}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 0)

	analyzed := []analyzer.AnalyzedTransaction{
		{
			Message:     &message.Message{Timestamp: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC), Sender: "102"},
			Transaction: &template.Transaction{Card: "9..1234", Converted: template.Amount{Value: 93719.33, Currency: "MDL"}, Operation: "Suplinire", Address: "Plata salariala"},
			Kind:        analyzer.KindIncome,
			PairIndex:   -1,
		},
	}

	result, err := syncer.Sync(analyzed)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if result.Synced != 1 {
		t.Errorf("Synced = %d, want 1", result.Synced)
	}
	if len(capturedTransactions) != 1 {
		t.Fatalf("Expected 1 transaction, got %d", len(capturedTransactions))
	}
	if capturedTransactions[0].Amount <= 0 {
		t.Errorf("Amount = %d, want positive (income)", capturedTransactions[0].Amount)
	}
	if capturedTransactions[0].TransferAccountID != "" {
		t.Errorf("TransferAccountID = %q, want empty for income", capturedTransactions[0].TransferAccountID)
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
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 0)

	analyzed := []analyzer.AnalyzedTransaction{
		{
			Message:     &message.Message{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC), Sender: "102"},
			Transaction: &template.Transaction{Card: "9..1111", Converted: template.Amount{Value: 100.00, Currency: "MDL"}, Operation: "Debitare", Address: "Coffee Shop"},
			Kind:        analyzer.KindPayment,
			PairIndex:   -1,
		},
	}

	result, err := syncer.Sync(analyzed)
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

	// Mapper only knows about the debit account.
	mapper := NewMapper([]YNABAccount{
		{YNABAccountID: "acc-debit", Last4: "1111"},
	}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 0)

	base := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	analyzed := []analyzer.AnalyzedTransaction{
		{
			Message:     &message.Message{Timestamp: base, Sender: "102"},
			Transaction: &template.Transaction{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
			Kind:        analyzer.KindTransfer,
			HasPair:     true,
			PairIndex:   1,
		},
		{
			Message:     &message.Message{Timestamp: base.Add(2 * time.Minute), Sender: "102"},
			Transaction: &template.Transaction{Card: "9..2222", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Suplinire"},
			Kind:        analyzer.KindCreditTransfer,
			HasPair:     true,
			PairIndex:   0,
		},
	}

	result, err := syncer.Sync(analyzed)
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

func TestSyncer_Sync_WarnsDuplicateImportIDs(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			resp := &CreateTransactionsResponse{}
			for _, tx := range transactions {
				resp.Data.DuplicateImportIDs = append(resp.Data.DuplicateImportIDs, tx.ImportID)
			}
			return resp, nil
		},
	}

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 0)

	analyzed := []analyzer.AnalyzedTransaction{
		{
			Message:     &message.Message{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)},
			Transaction: &template.Transaction{Card: "9..1234", Converted: template.Amount{Value: 10, Currency: "MDL"}, Operation: "Debitare"},
			Kind:        analyzer.KindPayment,
			PairIndex:   -1,
		},
	}

	result, err := syncer.Sync(analyzed)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if len(result.Warnings) != 1 {
		t.Fatalf("Warnings = %v, want 1 entry", result.Warnings)
	}
	if !strings.HasPrefix(result.Warnings[0], "Warning: YNAB skipped as duplicate: YNAB:") {
		t.Errorf("Warnings[0] = %q, want prefix \"Warning: YNAB skipped as duplicate: YNAB:\"", result.Warnings[0])
	}
	if result.Synced != 0 {
		t.Errorf("Synced = %d, want 0 (duplicate should not be recorded)", result.Synced)
	}

	allSynced, _ := store.GetAllSynced()
	if len(allSynced) != 0 {
		t.Errorf("store has %d records, want 0 (duplicate should not be stored)", len(allSynced))
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
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, true, 0)

	msg := &message.Message{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)}
	tx := &template.Transaction{
		Card:      "9..1234",
		Converted: template.Amount{Value: 100, Currency: "MDL"},
		Operation: "Debitare",
	}

	result, err := syncer.Sync(makeAnalyzed([]*message.Message{msg}, []*template.Transaction{tx}))
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
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 0)

	msg := &message.Message{Timestamp: time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)}
	tx := &template.Transaction{
		Card:      "9..1234",
		Converted: template.Amount{Value: 100, Currency: "MDL"},
		Operation: "Debitare",
	}

	result, err := syncer.Sync(makeAnalyzed([]*message.Message{msg}, []*template.Transaction{tx}))
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

func TestSyncer_Sync_DebitWaitWindow_SkipsRecentDebit(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			t.Error("should not sync debit within wait window")
			return &CreateTransactionsResponse{}, nil
		},
	}

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 6*time.Hour)

	analyzed := []analyzer.AnalyzedTransaction{
		{
			Message:     &message.Message{Timestamp: time.Now().Add(-1 * time.Hour)},
			Transaction: &template.Transaction{Card: "9..1234", Converted: template.Amount{Value: 100, Currency: "MDL"}, Operation: "Debitare"},
			Kind:        analyzer.KindPayment,
			PairIndex:   -1,
		},
	}

	result, err := syncer.Sync(analyzed)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (within wait window)", result.Skipped)
	}
}

func TestSyncer_Sync_DebitWaitWindow_SyncsExpiredDebit(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	var capturedTransactions []TransactionPayload
	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			capturedTransactions = transactions
			resp := &CreateTransactionsResponse{}
			for range transactions {
				resp.Data.TransactionIDs = append(resp.Data.TransactionIDs, "txn-1")
			}
			return resp, nil
		},
	}

	mapper := NewMapper([]YNABAccount{{YNABAccountID: "acc-1", Last4: "1234"}}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 6*time.Hour)

	analyzed := []analyzer.AnalyzedTransaction{
		{
			Message:     &message.Message{Timestamp: time.Now().Add(-8 * time.Hour)},
			Transaction: &template.Transaction{Card: "9..1234", Converted: template.Amount{Value: 100, Currency: "MDL"}, Operation: "Debitare"},
			Kind:        analyzer.KindPayment,
			PairIndex:   -1,
		},
	}

	result, err := syncer.Sync(analyzed)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result.Synced != 1 {
		t.Errorf("Synced = %d, want 1 (past wait window)", result.Synced)
	}
	if len(capturedTransactions) != 1 {
		t.Errorf("Expected 1 transaction sent to API, got %d", len(capturedTransactions))
	}
}

func TestSyncer_Sync_DebitWaitWindow_DoesNotAffectTransfers(t *testing.T) {
	store, _ := NewSyncStore(t.TempDir() + "/data.json")
	defer store.Close()

	var capturedTransactions []TransactionPayload
	client := &mockClient{
		createTransactionsFunc: func(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
			capturedTransactions = transactions
			resp := &CreateTransactionsResponse{}
			for range transactions {
				resp.Data.TransactionIDs = append(resp.Data.TransactionIDs, "txn-1")
			}
			return resp, nil
		},
	}

	mapper := NewMapper([]YNABAccount{
		{YNABAccountID: "acc-debit", Last4: "1111"},
		{YNABAccountID: "acc-credit", Last4: "2222"},
	}, nil)
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 6*time.Hour)

	// Both messages recent (within wait window), but already matched as transfer pair.
	base := time.Now().Add(-30 * time.Minute)
	analyzed := []analyzer.AnalyzedTransaction{
		{
			Message:     &message.Message{Timestamp: base},
			Transaction: &template.Transaction{Card: "9..1111", Converted: template.Amount{Value: 500, Currency: "MDL"}, Operation: "Debitare"},
			Kind:        analyzer.KindTransfer,
			HasPair:     true,
			PairIndex:   1,
		},
		{
			Message:     &message.Message{Timestamp: base.Add(5 * time.Minute)},
			Transaction: &template.Transaction{Card: "9..2222", Converted: template.Amount{Value: 500, Currency: "MDL"}, Operation: "Suplinire"},
			Kind:        analyzer.KindCreditTransfer,
			HasPair:     true,
			PairIndex:   0,
		},
	}

	result, err := syncer.Sync(analyzed)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result.Synced != 1 {
		t.Errorf("Synced = %d, want 1 (transfer synced immediately despite wait window)", result.Synced)
	}
	if len(capturedTransactions) != 1 {
		t.Errorf("Expected 1 transaction sent to API, got %d", len(capturedTransactions))
	}
	if capturedTransactions[0].TransferAccountID != "acc-credit" {
		t.Errorf("TransferAccountID = %q, want %q", capturedTransactions[0].TransferAccountID, "acc-credit")
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
	syncer := NewSyncer(store, client, mapper, "test-budget", startDate, false, 0)

	txDate := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	messages := []*message.Message{
		{Timestamp: txDate, Sender: "102"},
	}
	transactions := []*template.Transaction{
		{Card: "9..1234", Converted: template.Amount{Value: 100, Currency: "MDL"}, Operation: "Debitare"},
	}

	_, err := syncer.Sync(makeAnalyzed(messages, transactions))
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
