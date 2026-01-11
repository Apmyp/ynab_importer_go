package ynab

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_CreateTransactions_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/budgets/test-budget/transactions" {
			t.Errorf("Expected /v1/budgets/test-budget/transactions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("Expected Bearer test-api-key, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		response := CreateTransactionsResponse{
			Data: struct {
				TransactionIDs []string `json:"transaction_ids"`
				Transactions   []struct {
					ID       string `json:"id"`
					ImportID string `json:"import_id"`
				} `json:"transactions,omitempty"`
			}{
				TransactionIDs: []string{"txn-1", "txn-2"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &HTTPClient{
		baseURL:    server.URL + "/v1",
		apiKey:     []byte("test-api-key"),
		httpClient: server.Client(),
	}

	transactions := []TransactionPayload{
		{
			AccountID: "account-1",
			Date:      "2026-01-10",
			Amount:    -10000,
			PayeeName: "Test",
			Cleared:   "cleared",
			ImportID:  "YNAB:test1",
		},
		{
			AccountID: "account-1",
			Date:      "2026-01-11",
			Amount:    -20000,
			PayeeName: "Test2",
			Cleared:   "cleared",
			ImportID:  "YNAB:test2",
		},
	}

	response, err := client.CreateTransactions("test-budget", transactions)
	if err != nil {
		t.Fatalf("CreateTransactions() error = %v", err)
	}

	if len(response.Data.TransactionIDs) != 2 {
		t.Errorf("Expected 2 transaction IDs, got %d", len(response.Data.TransactionIDs))
	}
}

func TestClient_CreateTransactions_RateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := &HTTPClient{
		baseURL:    server.URL + "/v1",
		apiKey:     []byte("test-api-key"),
		httpClient: server.Client(),
	}

	transactions := []TransactionPayload{
		{AccountID: "account-1", Date: "2026-01-10", Amount: -10000},
	}

	_, err := client.CreateTransactions("test-budget", transactions)
	if err == nil {
		t.Fatal("CreateTransactions() should return error on 429")
	}
	if !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("Expected ErrRateLimitExceeded, got %v", err)
	}
}

func TestClient_CreateTransactions_ClientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		errorResp := ErrorResponse{
			Error: struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Detail string `json:"detail"`
			}{
				ID:     "400",
				Name:   "bad_request",
				Detail: "Invalid request",
			},
		}
		json.NewEncoder(w).Encode(errorResp)
	}))
	defer server.Close()

	client := &HTTPClient{
		baseURL:    server.URL + "/v1",
		apiKey:     []byte("test-api-key"),
		httpClient: server.Client(),
	}

	transactions := []TransactionPayload{
		{AccountID: "account-1", Date: "2026-01-10", Amount: -10000},
	}

	_, err := client.CreateTransactions("test-budget", transactions)
	if err == nil {
		t.Error("CreateTransactions() should return error for 400 status")
	}
}

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient("test-api-key")
	if client == nil {
		t.Error("NewHTTPClient() returned nil")
	}
	if string(client.apiKey) != "test-api-key" {
		t.Errorf("apiKey = %v, want test-api-key", string(client.apiKey))
	}
	if client.baseURL != "https://api.youneedabudget.com/v1" {
		t.Errorf("baseURL = %v, want https://api.youneedabudget.com/v1", client.baseURL)
	}
}

func TestHTTPClient_ClearAPIKey(t *testing.T) {
	client := NewHTTPClient("secret-key")

	if string(client.apiKey) != "secret-key" {
		t.Error("API key not set correctly")
	}

	client.ClearAPIKey()

	if client.apiKey != nil {
		t.Error("API key should be nil after ClearAPIKey()")
	}
}

func TestClient_CreateTransactions_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &HTTPClient{
		baseURL:    server.URL + "/v1",
		apiKey:     []byte("test-api-key"),
		httpClient: server.Client(),
	}

	transactions := []TransactionPayload{
		{AccountID: "account-1", Date: "2026-01-10", Amount: -10000},
	}

	_, err := client.CreateTransactions("test-budget", transactions)
	if err == nil {
		t.Error("CreateTransactions() should fail on 500 errors")
	}
}

func TestClient_GetAccounts_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/budgets/test-budget/accounts" {
			t.Errorf("Expected /v1/budgets/test-budget/accounts, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("Expected Bearer test-api-key, got %s", r.Header.Get("Authorization"))
		}

		response := GetAccountsResponse{
			Data: struct {
				Accounts []Account `json:"accounts"`
			}{
				Accounts: []Account{
					{ID: "account-1", Name: "Card 1234", Type: "checking", Balance: 100000, Closed: false, Deleted: false},
					{ID: "account-2", Name: "Card 5678", Type: "checking", Balance: 200000, Closed: false, Deleted: false},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &HTTPClient{
		baseURL:    server.URL + "/v1",
		apiKey:     []byte("test-api-key"),
		httpClient: server.Client(),
	}

	response, err := client.GetAccounts("test-budget")
	if err != nil {
		t.Fatalf("GetAccounts() error = %v", err)
	}

	if len(response.Data.Accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(response.Data.Accounts))
	}
}

func TestClient_GetAccounts_RateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := &HTTPClient{
		baseURL:    server.URL + "/v1",
		apiKey:     []byte("test-api-key"),
		httpClient: server.Client(),
	}

	_, err := client.GetAccounts("test-budget")
	if err == nil {
		t.Error("GetAccounts() should fail on rate limit")
	}
	if !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("Expected ErrRateLimitExceeded, got %v", err)
	}
}

func TestClient_GetAccounts_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &HTTPClient{
		baseURL:    server.URL + "/v1",
		apiKey:     []byte("test-api-key"),
		httpClient: server.Client(),
	}

	_, err := client.GetAccounts("test-budget")
	if err == nil {
		t.Error("GetAccounts() should fail on 500 errors")
	}
}

func TestClient_CreateAccount_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/budgets/test-budget/accounts" {
			t.Errorf("Expected /v1/budgets/test-budget/accounts, got %s", r.URL.Path)
		}

		var req CreateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.Account.Name != "Card 1234" {
			t.Errorf("Expected account name 'Card 1234', got %s", req.Account.Name)
		}

		response := CreateAccountResponse{
			Data: struct {
				Account Account `json:"account"`
			}{
				Account: Account{
					ID:   "new-account-id",
					Name: "Card 1234",
					Type: "checking",
				},
			},
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &HTTPClient{
		baseURL:    server.URL + "/v1",
		apiKey:     []byte("test-api-key"),
		httpClient: server.Client(),
	}

	payload := CreateAccountPayload{
		Name:    "Card 1234",
		Type:    "checking",
		Balance: 0,
	}

	response, err := client.CreateAccount("test-budget", payload)
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}

	if response.Data.Account.ID != "new-account-id" {
		t.Errorf("Expected account ID 'new-account-id', got %s", response.Data.Account.ID)
	}
}

func TestClient_CreateAccount_RateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := &HTTPClient{
		baseURL:    server.URL + "/v1",
		apiKey:     []byte("test-api-key"),
		httpClient: server.Client(),
	}

	payload := CreateAccountPayload{Name: "Card 1234", Type: "checking"}

	_, err := client.CreateAccount("test-budget", payload)
	if err == nil {
		t.Error("CreateAccount() should fail on rate limit")
	}
	if !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("Expected ErrRateLimitExceeded, got %v", err)
	}
}

func TestClient_CreateAccount_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &HTTPClient{
		baseURL:    server.URL + "/v1",
		apiKey:     []byte("test-api-key"),
		httpClient: server.Client(),
	}

	payload := CreateAccountPayload{Name: "Card 1234", Type: "checking"}

	_, err := client.CreateAccount("test-budget", payload)
	if err == nil {
		t.Error("CreateAccount() should fail on 500 errors")
	}
}
