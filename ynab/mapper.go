package ynab

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/apmyp/ynab_importer_go/message"
	"github.com/apmyp/ynab_importer_go/template"
)

type Mapper struct {
	accountsByLast4 map[string]string
	last4Regex      *regexp.Regexp
}

func NewMapper(accounts []YNABAccount) *Mapper {
	accountsByLast4 := make(map[string]string)
	for _, acc := range accounts {
		accountsByLast4[acc.Last4] = acc.YNABAccountID
	}

	return &Mapper{
		accountsByLast4: accountsByLast4,
		last4Regex:      regexp.MustCompile(`\d{4}$`),
	}
}

func (m *Mapper) MatchAccount(tx *template.Transaction) (string, error) {
	if tx.Card == "" {
		return "", errors.New("transaction has no card information")
	}

	// Extract last 4 digits from card field (e.g., "9..1234" -> "1234")
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

func (m *Mapper) GenerateImportID(msg *message.Message, tx *template.Transaction) string {
	// Format: timestamp:card:amount:payee
	data := fmt.Sprintf("%d:%s:%.2f:%s",
		msg.Timestamp.Unix(),
		tx.Card,
		tx.Converted.Value,
		tx.Address,
	)

	hash := sha256.Sum256([]byte(data))
	hashStr := hex.EncodeToString(hash[:8])

	return fmt.Sprintf("YNAB:%s", hashStr)
}

func (m *Mapper) MapTransaction(msg *message.Message, tx *template.Transaction) (*TransactionPayload, error) {
	accountID, err := m.MatchAccount(tx)
	if err != nil {
		return nil, err
	}

	importID := m.GenerateImportID(msg, tx)
	date := msg.Timestamp.Format("2006-01-02")

	// Amount in milliunits (multiply by 1000), negative for debits
	amountMilliunits := int64(tx.Converted.Value * 1000)
	if isDebit(tx.Operation) {
		amountMilliunits = -amountMilliunits
	}

	payeeName := tx.Address
	if payeeName == "" {
		payeeName = "Unknown"
	}

	memo := buildMemo(tx)

	return &TransactionPayload{
		AccountID: accountID,
		Date:      date,
		Amount:    amountMilliunits,
		PayeeName: payeeName,
		Memo:      memo,
		Cleared:   "cleared",
		ImportID:  importID,
	}, nil
}

func buildMemo(tx *template.Transaction) string {
	standardOperations := []string{
		"Tovary i uslugi",
		"Debitare",
		"Suplinire",
		"Tranzactie reusita",
	}

	standardStatuses := []string{
		"Odobrena",
		"",
	}

	operationIsStandard := false
	for _, stdOp := range standardOperations {
		if tx.Operation == stdOp {
			operationIsStandard = true
			break
		}
	}

	statusIsStandard := false
	for _, stdStatus := range standardStatuses {
		if tx.Status == stdStatus {
			statusIsStandard = true
			break
		}
	}

	if operationIsStandard && statusIsStandard {
		return ""
	}

	var memoParts []string

	if !operationIsStandard {
		memoParts = append(memoParts, tx.Operation)
	}

	if !statusIsStandard {
		memoParts = append(memoParts, tx.Status)
	}

	return strings.Join(memoParts, " - ")
}

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
