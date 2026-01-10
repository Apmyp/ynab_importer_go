// Package ynab handles interaction with YNAB API
package ynab

// Client handles communication with YNAB API
type Client struct {
	apiKey  string
	baseURL string
}

// NewClient creates a new YNAB API client
func NewClient() *Client {
	return &Client{baseURL: "https://api.youneedabudget.com/v1"}
}

// SetAPIKey sets the API key for authentication
func (c *Client) SetAPIKey(key string) {
	c.apiKey = key
}

// GetAPIKey returns the current API key
func (c *Client) GetAPIKey() string {
	return c.apiKey
}

// GetBaseURL returns the base API URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}
