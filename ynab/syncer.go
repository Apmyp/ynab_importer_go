package ynab

import (
	"fmt"
	"math"
	"regexp"
	"time"

	"github.com/apmyp/ynab_importer_go/message"
	"github.com/apmyp/ynab_importer_go/template"
)

type YNABClient interface {
	CreateTransactions(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error)
	GetAccounts(budgetID string) (*GetAccountsResponse, error)
	CreateAccount(budgetID string, payload CreateAccountPayload) (*CreateAccountResponse, error)
	GetTransactions(budgetID string, sinceDate string) (*GetTransactionsResponse, error)
	GetCategories(budgetID string) (*GetCategoriesResponse, error)
	DeleteTransaction(budgetID, transactionID string) error
}

type Syncer struct {
	store     *SyncStore
	client    YNABClient
	mapper    *Mapper
	budgetID  string
	startDate time.Time
	reimport  bool
}

type SyncResult struct {
	Total         int
	Synced        int
	Skipped       int
	Failed        []string
	Warnings      []string
	UnknownPayees []string
}

func NewSyncer(store *SyncStore, client YNABClient, mapper *Mapper, budgetID string, startDate time.Time, reimport bool) *Syncer {
	return &Syncer{
		store:     store,
		client:    client,
		mapper:    mapper,
		budgetID:  budgetID,
		startDate: startDate,
		reimport:  reimport,
	}
}

var last4Regex = regexp.MustCompile(`\d{4}$`)

func extractLast4(card string) string {
	return last4Regex.FindString(card)
}

func detectTransferPairs(messages []*message.Message, transactions []*template.Transaction) map[int]int {
	pairs := make(map[int]int)
	creditUsed := make(map[int]bool)

	for i, tx := range transactions {
		if !isDebit(tx.Operation) {
			continue
		}

		for j, candidate := range transactions {
			if i == j {
				continue
			}
			if isDebit(candidate.Operation) {
				continue
			}
			if creditUsed[j] {
				continue
			}

			diff := math.Abs(tx.Converted.Value - candidate.Converted.Value)
			if diff > 0.01 {
				continue
			}

			if extractLast4(tx.Card) == extractLast4(candidate.Card) {
				continue
			}

			timeDiff := messages[i].Timestamp.Sub(messages[j].Timestamp)
			if timeDiff < 0 {
				timeDiff = -timeDiff
			}
			if timeDiff > 5*time.Minute {
				continue
			}

			pairs[i] = j
			creditUsed[j] = true
			break
		}
	}

	return pairs
}

func (s *Syncer) Sync(messages []*message.Message, transactions []*template.Transaction) (*SyncResult, error) {
	result := &SyncResult{
		Total: len(transactions),
	}

	if len(messages) != len(transactions) {
		return nil, fmt.Errorf("messages and transactions length mismatch: %d vs %d", len(messages), len(transactions))
	}

	transferPairs := detectTransferPairs(messages, transactions)
	creditSideIndexes := make(map[int]bool)
	creditToDebit := make(map[int]int)
	for debitIdx, creditIdx := range transferPairs {
		creditSideIndexes[creditIdx] = true
		creditToDebit[creditIdx] = debitIdx
	}

	var toSync []TransactionPayload
	var toSyncImportIDs []string

	for i := 0; i < len(transactions); i++ {
		msg := messages[i]
		tx := transactions[i]

		if creditSideIndexes[i] {
			debitIdx := creditToDebit[i]
			debitMsg := messages[debitIdx]
			if !debitMsg.Timestamp.Before(s.startDate) {
				result.Skipped++
				continue
			}
			debitImportID := s.mapper.GenerateImportID(debitMsg, transactions[debitIdx])
			debitSynced, err := s.store.IsSynced(debitImportID)
			if err != nil {
				return nil, fmt.Errorf("failed to check debit sync status: %w", err)
			}
			if debitSynced {
				result.Skipped++
				continue
			}
		}

		if msg.Timestamp.Before(s.startDate) {
			result.Skipped++
			continue
		}

		importID := s.mapper.GenerateImportID(msg, tx)

		synced, err := s.store.IsSynced(importID)
		if err != nil {
			return nil, fmt.Errorf("failed to check sync status: %w", err)
		}
		if synced {
			result.Skipped++
			continue
		}

		payload, err := s.mapper.MapTransaction(msg, tx)
		if err != nil {
			result.Skipped++
			result.Failed = append(result.Failed, fmt.Sprintf("Failed to map: %v", err))
			continue
		}

		if creditIdx, isTransfer := transferPairs[i]; isTransfer {
			creditTx := transactions[creditIdx]
			creditAccountID, err := s.mapper.MatchAccount(creditTx)
			if err != nil {
				result.Skipped++
				result.Failed = append(result.Failed, fmt.Sprintf("Failed to match credit account: %v", err))
				continue
			}
			payload.TransferAccountID = creditAccountID
		}

		if payload.CategoryID == "" && payload.PayeeName != "" {
			if !containsString(result.UnknownPayees, payload.PayeeName) {
				result.UnknownPayees = append(result.UnknownPayees, payload.PayeeName)
			}
		}

		if s.reimport {
			payload.ImportID = ""
		}

		toSync = append(toSync, *payload)
		toSyncImportIDs = append(toSyncImportIDs, importID)
	}

	if len(toSync) == 0 {
		return result, nil
	}

	batchSize := 100
	for i := 0; i < len(toSync); i += batchSize {
		end := i + batchSize
		if end > len(toSync) {
			end = len(toSync)
		}

		batch := toSync[i:end]
		batchImportIDs := toSyncImportIDs[i:end]

		resp, err := s.client.CreateTransactions(s.budgetID, batch)
		if err != nil {
			return result, fmt.Errorf("failed to create transactions: %w", err)
		}

		for _, dupID := range resp.Data.DuplicateImportIDs {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Warning: YNAB skipped as duplicate: %s", dupID))
		}

		for j, importID := range batchImportIDs {
			record := &SyncRecord{
				ImportID:        importID,
				SyncedAt:        time.Now().UTC(),
				TransactionDate: batch[j].Date,
			}
			if err := s.store.RecordSync(record); err != nil {
				return result, fmt.Errorf("failed to record sync: %w", err)
			}
			result.Synced++
		}
	}

	return result, nil
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
