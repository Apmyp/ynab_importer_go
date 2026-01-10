package ynab

import (
	"errors"
	"strings"
	"testing"

	"github.com/apmyp/ynab_importer_go/config"
	"github.com/apmyp/ynab_importer_go/template"
)

func TestAccountManager_EnsureAccounts_AllAccountsExist(t *testing.T) {
	client := &mockClient{}
	manager := NewAccountManager(client)

	existingAccounts := []config.YNABAccount{
		{YNABAccountID: "acc-1", Last4: "1234"},
		{YNABAccountID: "acc-2", Last4: "5678"},
	}

	transactions := []*template.Transaction{
		{Card: "9..1234"},
		{Card: "*5678"},
	}

	result, err := manager.EnsureAccounts("test-budget", existingAccounts, transactions)
	if err != nil {
		t.Fatalf("EnsureAccounts() error = %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(result))
	}
}

func TestAccountManager_EnsureAccounts_FindsExistingAccount(t *testing.T) {
	client := &mockClient{
		getAccountsFunc: func(budgetID string) (*GetAccountsResponse, error) {
			return &GetAccountsResponse{
				Data: struct {
					Accounts []Account `json:"accounts"`
				}{
					Accounts: []Account{
						{ID: "found-acc", Name: "Card 9999", Type: "checking", Closed: false, Deleted: false},
					},
				},
			}, nil
		},
	}
	manager := NewAccountManager(client)

	existingAccounts := []config.YNABAccount{
		{YNABAccountID: "acc-1", Last4: "1234"},
	}

	transactions := []*template.Transaction{
		{Card: "9..1234"},
		{Card: "*9999"}, // This one needs to be found
	}

	result, err := manager.EnsureAccounts("test-budget", existingAccounts, transactions)
	if err != nil {
		t.Fatalf("EnsureAccounts() error = %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(result))
	}

	// Check that new account was added
	found := false
	for _, acc := range result {
		if acc.Last4 == "9999" && acc.YNABAccountID == "found-acc" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find account for card 9999")
	}
}

func TestAccountManager_EnsureAccounts_CreatesNewAccount(t *testing.T) {
	client := &mockClient{
		getAccountsFunc: func(budgetID string) (*GetAccountsResponse, error) {
			return &GetAccountsResponse{
				Data: struct {
					Accounts []Account `json:"accounts"`
				}{
					Accounts: []Account{}, // No existing accounts with 9999
				},
			}, nil
		},
		createAccountFunc: func(budgetID string, payload CreateAccountPayload) (*CreateAccountResponse, error) {
			if !strings.Contains(payload.Name, "9999") {
				t.Errorf("Expected account name to contain 9999, got %s", payload.Name)
			}
			if payload.Type != "checking" {
				t.Errorf("Expected account type checking, got %s", payload.Type)
			}
			return &CreateAccountResponse{
				Data: struct {
					Account Account `json:"account"`
				}{
					Account: Account{
						ID:   "new-acc",
						Name: payload.Name,
						Type: payload.Type,
					},
				},
			}, nil
		},
	}
	manager := NewAccountManager(client)

	existingAccounts := []config.YNABAccount{
		{YNABAccountID: "acc-1", Last4: "1234"},
	}

	transactions := []*template.Transaction{
		{Card: "9..1234"},
		{Card: "*9999"}, // This one needs to be created
	}

	result, err := manager.EnsureAccounts("test-budget", existingAccounts, transactions)
	if err != nil {
		t.Fatalf("EnsureAccounts() error = %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(result))
	}

	// Check that new account was created and added
	found := false
	for _, acc := range result {
		if acc.Last4 == "9999" && acc.YNABAccountID == "new-acc" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find newly created account for card 9999")
	}
}

func TestAccountManager_EnsureAccounts_GetAccountsFails(t *testing.T) {
	client := &mockClient{
		getAccountsFunc: func(budgetID string) (*GetAccountsResponse, error) {
			return nil, errors.New("API error")
		},
	}
	manager := NewAccountManager(client)

	existingAccounts := []config.YNABAccount{}
	transactions := []*template.Transaction{
		{Card: "*9999"},
	}

	_, err := manager.EnsureAccounts("test-budget", existingAccounts, transactions)
	if err == nil {
		t.Error("EnsureAccounts() should fail when GetAccounts fails")
	}
}

func TestAccountManager_EnsureAccounts_CreateAccountFails(t *testing.T) {
	client := &mockClient{
		getAccountsFunc: func(budgetID string) (*GetAccountsResponse, error) {
			return &GetAccountsResponse{
				Data: struct {
					Accounts []Account `json:"accounts"`
				}{
					Accounts: []Account{},
				},
			}, nil
		},
		createAccountFunc: func(budgetID string, payload CreateAccountPayload) (*CreateAccountResponse, error) {
			return nil, errors.New("API error")
		},
	}
	manager := NewAccountManager(client)

	existingAccounts := []config.YNABAccount{}
	transactions := []*template.Transaction{
		{Card: "*9999"},
	}

	_, err := manager.EnsureAccounts("test-budget", existingAccounts, transactions)
	if err == nil {
		t.Error("EnsureAccounts() should fail when CreateAccount fails")
	}
}

func TestAccountManager_EnsureAccounts_SkipsClosedAccounts(t *testing.T) {
	client := &mockClient{
		getAccountsFunc: func(budgetID string) (*GetAccountsResponse, error) {
			return &GetAccountsResponse{
				Data: struct {
					Accounts []Account `json:"accounts"`
				}{
					Accounts: []Account{
						{ID: "closed-acc", Name: "Card 9999", Type: "checking", Closed: true, Deleted: false},
						{ID: "open-acc", Name: "My Card 9999", Type: "checking", Closed: false, Deleted: false},
					},
				},
			}, nil
		},
	}
	manager := NewAccountManager(client)

	existingAccounts := []config.YNABAccount{}
	transactions := []*template.Transaction{
		{Card: "*9999"},
	}

	result, err := manager.EnsureAccounts("test-budget", existingAccounts, transactions)
	if err != nil {
		t.Fatalf("EnsureAccounts() error = %v", err)
	}

	// Should find open account, not closed one
	if len(result) != 1 {
		t.Errorf("Expected 1 account, got %d", len(result))
	}
	if result[0].YNABAccountID != "open-acc" {
		t.Errorf("Expected to use open account, got %s", result[0].YNABAccountID)
	}
}
