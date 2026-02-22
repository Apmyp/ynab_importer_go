package ynab

import "time"

type TransactionPayload struct {
	AccountID         string `json:"account_id"`
	Date              string `json:"date"`
	Amount            int64  `json:"amount"` // Milliunits (amount * 1000)
	PayeeName         string `json:"payee_name,omitempty"`
	Memo              string `json:"memo,omitempty"`
	Cleared           string `json:"cleared"`
	ImportID          string `json:"import_id,omitempty"`
	TransferAccountID string `json:"transfer_account_id,omitempty"`
	CategoryID        string `json:"category_id,omitempty"`
}

type SyncRecord struct {
	ImportID        string    `json:"import_id"`
	SyncedAt        time.Time `json:"synced_at"`
	TransactionDate string    `json:"transaction_date,omitempty"`
}

type YNABAccount struct {
	YNABAccountID string `json:"ynab_account_id"`
	Last4         string `json:"last4"`
}

type CreateTransactionsRequest struct {
	Transactions []TransactionPayload `json:"transactions"`
}

type CreateTransactionsResponse struct {
	Data struct {
		TransactionIDs []string `json:"transaction_ids"`
		Transactions   []struct {
			ID       string `json:"id"`
			ImportID string `json:"import_id"`
		} `json:"transactions,omitempty"`
	} `json:"data"`
}

type ErrorResponse struct {
	Error struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Detail string `json:"detail"`
	} `json:"error"`
}

type Account struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Balance int64  `json:"balance"`
	Closed  bool   `json:"closed"`
	Deleted bool   `json:"deleted"`
}

type GetAccountsResponse struct {
	Data struct {
		Accounts []Account `json:"accounts"`
	} `json:"data"`
}

type CreateAccountPayload struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Balance int64  `json:"balance"`
}

type CreateAccountRequest struct {
	Account CreateAccountPayload `json:"account"`
}

type CreateAccountResponse struct {
	Data struct {
		Account Account `json:"account"`
	} `json:"data"`
}

type Budget struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type GetBudgetsResponse struct {
	Data struct {
		Budgets []Budget `json:"budgets"`
	} `json:"data"`
}

type TransactionDetail struct {
	ID           string `json:"id"`
	Date         string `json:"date"`
	PayeeName    string `json:"payee_name"`
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
	Memo         string `json:"memo"`
	Amount       int64  `json:"amount"`
	Deleted      bool   `json:"deleted"`
}

type GetTransactionsResponse struct {
	Data struct {
		Transactions []TransactionDetail `json:"transactions"`
	} `json:"data"`
}

type CategoryItem struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Deleted bool   `json:"deleted"`
	Hidden  bool   `json:"hidden"`
}

type CategoryGroup struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Categories []CategoryItem `json:"categories"`
}

type GetCategoriesResponse struct {
	Data struct {
		CategoryGroups []CategoryGroup `json:"category_groups"`
	} `json:"data"`
}
