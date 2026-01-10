package ynab

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_CreateTransactions_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
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

		// Return success response
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
		apiKey:     "test-api-key",
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
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			// Return 429 for first 2 attempts
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		// Success on 3rd attempt
		response := CreateTransactionsResponse{
			Data: struct {
				TransactionIDs []string `json:"transaction_ids"`
				Transactions   []struct {
					ID       string `json:"id"`
					ImportID string `json:"import_id"`
				} `json:"transactions,omitempty"`
			}{
				TransactionIDs: []string{"txn-1"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &HTTPClient{
		baseURL:    server.URL + "/v1",
		apiKey:     "test-api-key",
		httpClient: server.Client(),
		retryDelay: 10 * time.Millisecond, // Short delay for testing
		maxRetries: 3,
	}

	transactions := []TransactionPayload{
		{AccountID: "account-1", Date: "2026-01-10", Amount: -10000},
	}

	_, err := client.CreateTransactions("test-budget", transactions)
	if err != nil {
		t.Fatalf("CreateTransactions() should succeed after retries, error = %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestClient_CreateTransactions_PermanentError(t *testing.T) {
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
		apiKey:     "test-api-key",
		httpClient: server.Client(),
		maxRetries: 3,
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
	if client.apiKey != "test-api-key" {
		t.Errorf("apiKey = %v, want test-api-key", client.apiKey)
	}
	if client.baseURL != "https://api.youneedabudget.com/v1" {
		t.Errorf("baseURL = %v, want https://api.youneedabudget.com/v1", client.baseURL)
	}
}

func TestClient_CreateTransactions_ServerError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &HTTPClient{
		baseURL:    server.URL + "/v1",
		apiKey:     "test-api-key",
		httpClient: server.Client(),
		retryDelay: 10 * time.Millisecond,
		maxRetries: 3,
	}

	transactions := []TransactionPayload{
		{AccountID: "account-1", Date: "2026-01-10", Amount: -10000},
	}

	_, err := client.CreateTransactions("test-budget", transactions)
	if err == nil {
		t.Error("CreateTransactions() should fail after max retries on 500 errors")
	}

	if attempts != 4 { // initial + 3 retries
		t.Errorf("Expected 4 attempts, got %d", attempts)
	}
}

func TestClient_CreateTransactions_UnexpectedStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // 418
	}))
	defer server.Close()

	client := &HTTPClient{
		baseURL:    server.URL + "/v1",
		apiKey:     "test-api-key",
		httpClient: server.Client(),
		retryDelay: 10 * time.Millisecond,
		maxRetries: 3,
	}

	transactions := []TransactionPayload{
		{AccountID: "account-1", Date: "2026-01-10", Amount: -10000},
	}

	_, err := client.CreateTransactions("test-budget", transactions)
	if err == nil {
		t.Error("CreateTransactions() should return error for unexpected status code")
	}
}
