package ynab

import "testing"

func TestNewClient(t *testing.T) {
	c := NewClient()
	if c == nil {
		t.Error("NewClient() should return a non-nil Client")
	}
	if c.GetBaseURL() != "https://api.youneedabudget.com/v1" {
		t.Errorf("NewClient() default base URL = %q, want %q", c.GetBaseURL(), "https://api.youneedabudget.com/v1")
	}
}

func TestSetAPIKey(t *testing.T) {
	c := NewClient()
	testKey := "test-api-key-123"
	c.SetAPIKey(testKey)
	if c.GetAPIKey() != testKey {
		t.Errorf("GetAPIKey() = %q, want %q", c.GetAPIKey(), testKey)
	}
}

func TestGetAPIKey(t *testing.T) {
	c := NewClient()
	if c.GetAPIKey() != "" {
		t.Errorf("GetAPIKey() on new Client = %q, want empty string", c.GetAPIKey())
	}
}

func TestGetBaseURL(t *testing.T) {
	c := NewClient()
	if c.GetBaseURL() == "" {
		t.Error("GetBaseURL() should return non-empty default")
	}
}
