package ynab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient handles HTTP communication with YNAB API
type HTTPClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	retryDelay time.Duration
	maxRetries int
}

// NewHTTPClient creates a new HTTP client for YNAB API
func NewHTTPClient(apiKey string) *HTTPClient {
	return &HTTPClient{
		baseURL:    "https://api.youneedabudget.com/v1",
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		retryDelay: 2 * time.Second,
		maxRetries: 3,
	}
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

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			time.Sleep(c.retryDelay * time.Duration(attempt))
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			continue
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		// Handle rate limiting - retry
		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limit exceeded (429)")
			continue
		}

		// Handle server errors - retry
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		// Handle client errors - don't retry
		if resp.StatusCode >= 400 {
			var errorResp ErrorResponse
			if err := json.Unmarshal(body, &errorResp); err != nil {
				return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
			}
			return nil, fmt.Errorf("API error %d: %s - %s", resp.StatusCode, errorResp.Error.Name, errorResp.Error.Detail)
		}

		// Success
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			var response CreateTransactionsResponse
			if err := json.Unmarshal(body, &response); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}
			return &response, nil
		}

		lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// GetAccounts fetches all accounts for a budget
func (c *HTTPClient) GetAccounts(budgetID string) (*GetAccountsResponse, error) {
	url := fmt.Sprintf("%s/budgets/%s/accounts", c.baseURL, budgetID)

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay * time.Duration(attempt))
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			continue
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limit exceeded (429)")
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode >= 400 {
			var errorResp ErrorResponse
			if err := json.Unmarshal(body, &errorResp); err != nil {
				return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
			}
			return nil, fmt.Errorf("API error %d: %s - %s", resp.StatusCode, errorResp.Error.Name, errorResp.Error.Detail)
		}

		if resp.StatusCode == http.StatusOK {
			var response GetAccountsResponse
			if err := json.Unmarshal(body, &response); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}
			return &response, nil
		}

		lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
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

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay * time.Duration(attempt))
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			continue
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limit exceeded (429)")
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode >= 400 {
			var errorResp ErrorResponse
			if err := json.Unmarshal(body, &errorResp); err != nil {
				return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
			}
			return nil, fmt.Errorf("API error %d: %s - %s", resp.StatusCode, errorResp.Error.Name, errorResp.Error.Detail)
		}

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			var response CreateAccountResponse
			if err := json.Unmarshal(body, &response); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}
			return &response, nil
		}

		lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
