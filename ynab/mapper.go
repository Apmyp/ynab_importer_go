package ynab

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/apmyp/ynab_importer_go/bagoup"
	"github.com/apmyp/ynab_importer_go/template"
)

// Mapper handles transaction conversion and account matching
type Mapper struct {
	accountsByLast4 map[string]string // last4 -> ynab_account_id
	last4Regex      *regexp.Regexp
}

// NewMapper creates a new Mapper with account mappings
func NewMapper(accounts []YNABAccount) *Mapper {
	accountsByLast4 := make(map[string]string)
	for _, acc := range accounts {
		accountsByLast4[acc.Last4] = acc.YNABAccountID
	}

	return &Mapper{
		accountsByLast4: accountsByLast4,
		last4Regex:      regexp.MustCompile(`\d{4}$`), // Last 4 digits at end of string
	}
}

// MatchAccount finds the YNAB account ID for a transaction based on card last4
func (m *Mapper) MatchAccount(tx *template.Transaction) (string, error) {
	if tx.Card == "" {
		return "", errors.New("transaction has no card information")
	}

	// Extract last 4 digits from card field (e.g., "9..1234" -> "1234", "*1234" -> "1234")
	matches := m.last4Regex.FindString(tx.Card)
	if matches == "" {
		return "", fmt.Errorf("could not extract last4 from card: %s", tx.Card)
	}

	last4 := matches
	accountID, found := m.accountsByLast4[last4]
	if !found {
		return "", fmt.Errorf("no account found for card ending in %s", last4)
	}

	return accountID, nil
}

// GenerateImportID creates a unique, deterministic import ID for YNAB deduplication
func (m *Mapper) GenerateImportID(msg *bagoup.Message, tx *template.Transaction) string {
	// Create a deterministic string from transaction key fields
	// Format: timestamp:card:amount:payee
	data := fmt.Sprintf("%d:%s:%.2f:%s",
		msg.Timestamp.Unix(),
		tx.Card,
		tx.Converted.Value,
		tx.Address,
	)

	// Hash it to create a consistent, short ID
	hash := sha256.Sum256([]byte(data))
	hashStr := hex.EncodeToString(hash[:8]) // Use first 8 bytes (16 hex chars)

	// YNAB import_id format: YNAB:hash
	return fmt.Sprintf("YNAB:%s", hashStr)
}

// MapTransaction converts a parsed transaction to YNAB format
func (m *Mapper) MapTransaction(msg *bagoup.Message, tx *template.Transaction) (*TransactionPayload, error) {
	// Match account
	accountID, err := m.MatchAccount(tx)
	if err != nil {
		return nil, err
	}

	// Generate import ID
	importID := m.GenerateImportID(msg, tx)

	// Date in YYYY-MM-DD format
	date := msg.Timestamp.Format("2006-01-02")

	// Amount in milliunits (multiply by 1000 and convert to int)
	// Debitare/spending operations should be negative
	amountMilliunits := int64(tx.Converted.Value * 1000)
	if isDebit(tx.Operation) {
		amountMilliunits = -amountMilliunits
	}

	// Payee name from address field
	payeeName := tx.Address
	if payeeName == "" {
		payeeName = "Unknown"
	}

	// Memo - only include unique/important information
	memo := buildMemo(tx)

	return &TransactionPayload{
		AccountID: accountID,
		Date:      date,
		Amount:    amountMilliunits,
		PayeeName: payeeName,
		Memo:      memo,
		Cleared:   "cleared", // Bank transactions are cleared
		ImportID:  importID,
	}, nil
}

// buildMemo creates memo field, only including unique/important information
// Returns empty string for standard repetitive messages
func buildMemo(tx *template.Transaction) string {
	// Standard operations that don't need to be in memo
	standardOperations := []string{
		"Tovary i uslugi",
		"Debitare",
		"Suplinire",
		"Tranzactie reusita",
	}

	// Standard statuses that don't need to be in memo
	standardStatuses := []string{
		"Odobrena",
		"",
	}

	// Check if operation is standard
	operationIsStandard := false
	for _, stdOp := range standardOperations {
		if tx.Operation == stdOp {
			operationIsStandard = true
			break
		}
	}

	// Check if status is standard
	statusIsStandard := false
	for _, stdStatus := range standardStatuses {
		if tx.Status == stdStatus {
			statusIsStandard = true
			break
		}
	}

	// If both operation and status are standard, return empty memo
	if operationIsStandard && statusIsStandard {
		return ""
	}

	// Build memo with only non-standard parts
	var memoParts []string

	if !operationIsStandard {
		memoParts = append(memoParts, tx.Operation)
	}

	if !statusIsStandard {
		memoParts = append(memoParts, tx.Status)
	}

	return strings.Join(memoParts, " - ")
}

// isDebit returns true if the operation represents a debit/spending
func isDebit(operation string) bool {
	debitOperations := []string{
		"Debitare",
		"Tovary i uslugi",
		"Tranzactie reusita",
	}

	for _, op := range debitOperations {
		if strings.Contains(operation, op) {
			return true
		}
	}

	return false
}
