package ynab

import (
	"errors"
	"testing"
	"time"
)

type mockYNABClientForCategorizer struct {
	getTransactionsFunc func(budgetID, sinceDate string) (*GetTransactionsResponse, error)
	getCategoriesFunc   func(budgetID string) (*GetCategoriesResponse, error)
}

func (m *mockYNABClientForCategorizer) CreateTransactions(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
	return &CreateTransactionsResponse{}, nil
}
func (m *mockYNABClientForCategorizer) GetAccounts(budgetID string) (*GetAccountsResponse, error) {
	return &GetAccountsResponse{}, nil
}
func (m *mockYNABClientForCategorizer) CreateAccount(budgetID string, payload CreateAccountPayload) (*CreateAccountResponse, error) {
	return &CreateAccountResponse{}, nil
}
func (m *mockYNABClientForCategorizer) GetTransactions(budgetID, sinceDate string) (*GetTransactionsResponse, error) {
	if m.getTransactionsFunc != nil {
		return m.getTransactionsFunc(budgetID, sinceDate)
	}
	return &GetTransactionsResponse{}, nil
}
func (m *mockYNABClientForCategorizer) GetCategories(budgetID string) (*GetCategoriesResponse, error) {
	if m.getCategoriesFunc != nil {
		return m.getCategoriesFunc(budgetID)
	}
	return &GetCategoriesResponse{}, nil
}
func (m *mockYNABClientForCategorizer) DeleteTransaction(budgetID, transactionID string) error {
	return nil
}

func TestSeedCategoriesFromYNAB_PopulatesStore(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)

	client := &mockYNABClientForCategorizer{
		getTransactionsFunc: func(budgetID, sinceDate string) (*GetTransactionsResponse, error) {
			resp := &GetTransactionsResponse{}
			resp.Data.Transactions = []TransactionDetail{
				{ID: "txn-1", PayeeName: "Coffee Shop", CategoryID: "cat-food", CategoryName: "Food"},
				{ID: "txn-2", PayeeName: "Supermarket", CategoryID: "cat-grocery", CategoryName: "Groceries"},
			}
			return resp, nil
		},
	}

	err := SeedCategoriesFromYNAB(client, "budget-1", store)
	if err != nil {
		t.Fatalf("SeedCategoriesFromYNAB() error = %v", err)
	}

	if store.Get("Coffee Shop") != "cat-food" {
		t.Errorf("Get(Coffee Shop) = %q, want cat-food", store.Get("Coffee Shop"))
	}
	if store.Get("Supermarket") != "cat-grocery" {
		t.Errorf("Get(Supermarket) = %q, want cat-grocery", store.Get("Supermarket"))
	}
}

func TestSeedCategoriesFromYNAB_SkipsEmptyPayee(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)

	client := &mockYNABClientForCategorizer{
		getTransactionsFunc: func(budgetID, sinceDate string) (*GetTransactionsResponse, error) {
			resp := &GetTransactionsResponse{}
			resp.Data.Transactions = []TransactionDetail{
				{ID: "txn-1", PayeeName: "", CategoryID: "cat-food", CategoryName: "Food"},
				{ID: "txn-2", PayeeName: "Coffee Shop", CategoryID: "cat-food", CategoryName: "Food"},
			}
			return resp, nil
		},
	}

	if err := SeedCategoriesFromYNAB(client, "budget-1", store); err != nil {
		t.Fatalf("SeedCategoriesFromYNAB() error = %v", err)
	}

	all, _ := store.All()
	if len(all) != 1 {
		t.Errorf("Expected 1 entry (empty payee skipped), got %d", len(all))
	}
}

func TestSeedCategoriesFromYNAB_SkipsEmptyCategory(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)

	client := &mockYNABClientForCategorizer{
		getTransactionsFunc: func(budgetID, sinceDate string) (*GetTransactionsResponse, error) {
			resp := &GetTransactionsResponse{}
			resp.Data.Transactions = []TransactionDetail{
				{ID: "txn-1", PayeeName: "Coffee Shop", CategoryID: "", CategoryName: ""},
				{ID: "txn-2", PayeeName: "Supermarket", CategoryID: "cat-grocery", CategoryName: "Groceries"},
			}
			return resp, nil
		},
	}

	if err := SeedCategoriesFromYNAB(client, "budget-1", store); err != nil {
		t.Fatalf("SeedCategoriesFromYNAB() error = %v", err)
	}

	all, _ := store.All()
	if len(all) != 1 {
		t.Errorf("Expected 1 entry (empty category skipped), got %d", len(all))
	}
}

func TestSeedCategoriesFromYNAB_SkipsInflowCategory(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)

	client := &mockYNABClientForCategorizer{
		getTransactionsFunc: func(budgetID, sinceDate string) (*GetTransactionsResponse, error) {
			resp := &GetTransactionsResponse{}
			resp.Data.Transactions = []TransactionDetail{
				{ID: "txn-1", PayeeName: "Employer", CategoryID: "cat-inflow", CategoryName: "Inflow: Ready to Assign"},
				{ID: "txn-2", PayeeName: "Coffee Shop", CategoryID: "cat-food", CategoryName: "Food"},
			}
			return resp, nil
		},
	}

	if err := SeedCategoriesFromYNAB(client, "budget-1", store); err != nil {
		t.Fatalf("SeedCategoriesFromYNAB() error = %v", err)
	}

	all, _ := store.All()
	if len(all) != 1 {
		t.Errorf("Expected 1 entry (Inflow skipped), got %d: %v", len(all), all)
	}
	if store.Get("Employer") != "" {
		t.Errorf("Employer should be skipped (Inflow category)")
	}
}

func TestSeedCategoriesFromYNAB_RequestsLast12Months(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)

	var capturedSinceDate string
	client := &mockYNABClientForCategorizer{
		getTransactionsFunc: func(budgetID, sinceDate string) (*GetTransactionsResponse, error) {
			capturedSinceDate = sinceDate
			return &GetTransactionsResponse{}, nil
		},
	}

	if err := SeedCategoriesFromYNAB(client, "budget-1", store); err != nil {
		t.Fatalf("SeedCategoriesFromYNAB() error = %v", err)
	}

	if capturedSinceDate == "" {
		t.Error("Expected a since_date to be passed, got empty string")
	}
}

func TestSeedCategoriesFromYNAB_SkipsAPICallWhenFresh(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)
	_ = store.MarkSeeded()

	callCount := 0
	client := &mockYNABClientForCategorizer{
		getTransactionsFunc: func(budgetID, sinceDate string) (*GetTransactionsResponse, error) {
			callCount++
			return &GetTransactionsResponse{}, nil
		},
	}

	if err := SeedCategoriesFromYNAB(client, "budget-1", store); err != nil {
		t.Fatalf("SeedCategoriesFromYNAB() error = %v", err)
	}

	if callCount != 0 {
		t.Errorf("Expected 0 API calls when cache is fresh, got %d", callCount)
	}
}

func TestSeedCategoriesFromYNAB_MarksSeededAfterSuccess(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)

	client := &mockYNABClientForCategorizer{}

	if err := SeedCategoriesFromYNAB(client, "budget-1", store); err != nil {
		t.Fatalf("SeedCategoriesFromYNAB() error = %v", err)
	}

	fresh, err := store.IsSeededRecently(24 * time.Hour)
	if err != nil {
		t.Fatalf("IsSeededRecently() error = %v", err)
	}
	if !fresh {
		t.Error("Expected store to be marked as seeded after successful call")
	}
}

func TestSeedCategoriesFromYNAB_ReturnsErrorOnClientFailure(t *testing.T) {
	filePath := t.TempDir() + "/data.json"
	store := NewCategoryStore(filePath)

	client := &mockYNABClientForCategorizer{
		getTransactionsFunc: func(budgetID, sinceDate string) (*GetTransactionsResponse, error) {
			return nil, errors.New("API error")
		},
	}

	err := SeedCategoriesFromYNAB(client, "budget-1", store)
	if err == nil {
		t.Error("SeedCategoriesFromYNAB() should return error when client fails")
	}
}
