package ynab

import "time"

// TransactionPayload represents a YNAB transaction for API requests
type TransactionPayload struct {
	AccountID string `json:"account_id"`
	Date      string `json:"date"`   // YYYY-MM-DD format
	Amount    int64  `json:"amount"` // Milliunits (amount * 1000)
	PayeeName string `json:"payee_name,omitempty"`
	Memo      string `json:"memo,omitempty"`
	Cleared   string `json:"cleared"` // "cleared", "uncleared", "reconciled"
	ImportID  string `json:"import_id,omitempty"`
}

// SyncRecord tracks transactions that have been synced to YNAB
type SyncRecord struct {
	ImportID string    `json:"import_id"`
	SyncedAt time.Time `json:"synced_at"`
}

// YNABAccount maps a card's last 4 digits to a YNAB account ID
type YNABAccount struct {
	YNABAccountID string `json:"ynab_account_id"`
	Last4         string `json:"last4"`
}

// CreateTransactionsRequest is the request body for creating transactions
type CreateTransactionsRequest struct {
	Transactions []TransactionPayload `json:"transactions"`
}

// CreateTransactionsResponse is the response from creating transactions
type CreateTransactionsResponse struct {
	Data struct {
		TransactionIDs []string `json:"transaction_ids"`
		Transactions   []struct {
			ID       string `json:"id"`
			ImportID string `json:"import_id"`
		} `json:"transactions,omitempty"`
	} `json:"data"`
}

// ErrorResponse represents an error from the YNAB API
type ErrorResponse struct {
	Error struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Detail string `json:"detail"`
	} `json:"error"`
}
