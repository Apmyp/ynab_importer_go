package ynab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrRateLimitExceeded is returned when YNAB API returns 429
var ErrRateLimitExceeded = fmt.Errorf("YNAB rate limit exceeded (429)")

// HTTPClient handles HTTP communication with YNAB API
type HTTPClient struct {
	baseURL    string
	apiKey     []byte
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTP client for YNAB API
func NewHTTPClient(apiKey string) *HTTPClient {
	return &HTTPClient{
		baseURL:    "https://api.youneedabudget.com/v1",
		apiKey:     []byte(apiKey),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ClearAPIKey zeros out the API key in memory for security
func (c *HTTPClient) ClearAPIKey() {
	for i := range c.apiKey {
		c.apiKey[i] = 0
	}
	c.apiKey = nil
}

// doRequest performs an HTTP request and handles common error cases
func (c *HTTPClient) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", "Bearer "+string(c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimitExceeded
	}

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("YNAB server error: %d", resp.StatusCode)
	}

	if resp.StatusCode >= 400 {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("YNAB API error %d: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("YNAB API error %d: %s - %s", resp.StatusCode, errorResp.Error.Name, errorResp.Error.Detail)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return body, nil
}

// CreateTransactions sends transactions to YNAB API
func (c *HTTPClient) CreateTransactions(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error) {
	url := fmt.Sprintf("%s/budgets/%s/transactions", c.baseURL, budgetID)

	requestBody := CreateTransactionsRequest{
		Transactions: transactions,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response CreateTransactionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &response, nil
}

// GetAccounts fetches all accounts for a budget
func (c *HTTPClient) GetAccounts(budgetID string) (*GetAccountsResponse, error) {
	url := fmt.Sprintf("%s/budgets/%s/accounts", c.baseURL, budgetID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response GetAccountsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &response, nil
}

// GetBudgets fetches all budgets
func (c *HTTPClient) GetBudgets() (*GetBudgetsResponse, error) {
	url := fmt.Sprintf("%s/budgets", c.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response GetBudgetsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &response, nil
}

// CreateAccount creates a new account in YNAB
func (c *HTTPClient) CreateAccount(budgetID string, payload CreateAccountPayload) (*CreateAccountResponse, error) {
	url := fmt.Sprintf("%s/budgets/%s/accounts", c.baseURL, budgetID)

	requestBody := CreateAccountRequest{
		Account: payload,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response CreateAccountResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &response, nil
}
