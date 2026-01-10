package ynab

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/apmyp/ynab_importer_go/config"
	"github.com/apmyp/ynab_importer_go/template"
)

// AccountManager handles account matching and creation
type AccountManager struct {
	client     YNABClient
	last4Regex *regexp.Regexp
}

// NewAccountManager creates a new AccountManager
func NewAccountManager(client YNABClient) *AccountManager {
	return &AccountManager{
		client:     client,
		last4Regex: regexp.MustCompile(`\d{4}$`),
	}
}

// EnsureAccounts ensures all transactions have corresponding YNAB accounts
// It checks existing mappings, finds accounts by name, or creates new accounts
func (am *AccountManager) EnsureAccounts(
	budgetID string,
	existingAccounts []config.YNABAccount,
	transactions []*template.Transaction,
) ([]config.YNABAccount, error) {
	// Build map of existing last4 -> account ID
	accountMap := make(map[string]string)
	for _, acc := range existingAccounts {
		accountMap[acc.Last4] = acc.YNABAccountID
	}

	// Extract unique last4s from transactions
	uniqueLast4s := am.extractUniqueLast4s(transactions)

	// Find unmapped last4s
	var unmappedLast4s []string
	for _, last4 := range uniqueLast4s {
		if _, exists := accountMap[last4]; !exists {
			unmappedLast4s = append(unmappedLast4s, last4)
		}
	}

	// If all accounts are mapped, return existing accounts
	if len(unmappedLast4s) == 0 {
		return existingAccounts, nil
	}

	// Get all accounts from YNAB
	resp, err := am.client.GetAccounts(budgetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get YNAB accounts: %w", err)
	}

	// Try to match unmapped last4s with existing YNAB accounts
	result := make([]config.YNABAccount, len(existingAccounts))
	copy(result, existingAccounts)

	for _, last4 := range unmappedLast4s {
		// Try to find existing account
		var foundAccount *Account
		for i := range resp.Data.Accounts {
			acc := &resp.Data.Accounts[i]
			// Skip closed or deleted accounts
			if acc.Closed || acc.Deleted {
				continue
			}
			// Check if account name contains the last4
			if strings.Contains(acc.Name, last4) {
				foundAccount = acc
				break
			}
		}

		if foundAccount != nil {
			// Found existing account
			result = append(result, config.YNABAccount{
				YNABAccountID: foundAccount.ID,
				Last4:         last4,
			})
		} else {
			// Need to create new account
			payload := CreateAccountPayload{
				Name:    fmt.Sprintf("Card %s", last4),
				Type:    "checking",
				Balance: 0,
			}

			createResp, err := am.client.CreateAccount(budgetID, payload)
			if err != nil {
				return nil, fmt.Errorf("failed to create account for card %s: %w", last4, err)
			}

			result = append(result, config.YNABAccount{
				YNABAccountID: createResp.Data.Account.ID,
				Last4:         last4,
			})
		}
	}

	return result, nil
}

// extractUniqueLast4s extracts unique last4 card numbers from transactions
func (am *AccountManager) extractUniqueLast4s(transactions []*template.Transaction) []string {
	last4Set := make(map[string]bool)
	var result []string

	for _, tx := range transactions {
		if tx.Card == "" {
			continue
		}

		// Extract last 4 digits
		matches := am.last4Regex.FindString(tx.Card)
		if matches == "" {
			continue
		}

		last4 := matches
		if !last4Set[last4] {
			last4Set[last4] = true
			result = append(result, last4)
		}
	}

	return result
}
