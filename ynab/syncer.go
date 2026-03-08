package ynab

import (
	"fmt"
	"time"

	"github.com/apmyp/ynab_importer_go/analyzer"
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
	store           *SyncStore
	client          YNABClient
	mapper          *Mapper
	budgetID        string
	startDate       time.Time
	reimport        bool
	debitWaitWindow time.Duration
}

type SyncResult struct {
	Total         int
	Synced        int
	Skipped       int
	Failed   []string
	Warnings []string
}

func NewSyncer(store *SyncStore, client YNABClient, mapper *Mapper, budgetID string, startDate time.Time, reimport bool, debitWaitWindow time.Duration) *Syncer {
	return &Syncer{
		store:           store,
		client:          client,
		mapper:          mapper,
		budgetID:        budgetID,
		startDate:       startDate,
		reimport:        reimport,
		debitWaitWindow: debitWaitWindow,
	}
}

func (s *Syncer) Sync(analyzed []analyzer.AnalyzedTransaction) (*SyncResult, error) {
	result := &SyncResult{
		Total: len(analyzed),
	}

	var toSync []TransactionPayload
	var toSyncImportIDs []string

	for _, at := range analyzed {
		if at.Kind == analyzer.KindCreditTransfer {
			result.Skipped++
			continue
		}

		msg := at.Message
		tx := at.Transaction

		if msg.Timestamp.Before(s.startDate) {
			result.Skipped++
			continue
		}

		if s.debitWaitWindow > 0 && at.Kind == analyzer.KindPayment && isDebit(tx.Operation) {
			if time.Since(msg.Timestamp) < s.debitWaitWindow {
				result.Skipped++
				continue
			}
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

		if at.Kind == analyzer.KindTransfer && at.HasPair {
			creditTx := analyzed[at.PairIndex].Transaction
			creditAccountID, err := s.mapper.MatchAccount(creditTx)
			if err != nil {
				result.Skipped++
				result.Failed = append(result.Failed, fmt.Sprintf("Failed to match credit account: %v", err))
				continue
			}
			payload.TransferAccountID = creditAccountID
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

		dupSet := make(map[string]bool)
		for _, dupID := range resp.Data.DuplicateImportIDs {
			dupSet[dupID] = true
			result.Warnings = append(result.Warnings, fmt.Sprintf("Warning: YNAB skipped as duplicate: %s", dupID))
		}

		for j, importID := range batchImportIDs {
			if dupSet[importID] {
				continue
			}
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
