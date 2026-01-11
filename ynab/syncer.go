package ynab

import (
	"fmt"
	"time"

	"github.com/apmyp/ynab_importer_go/message"
	"github.com/apmyp/ynab_importer_go/template"
)

type YNABClient interface {
	CreateTransactions(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error)
	GetAccounts(budgetID string) (*GetAccountsResponse, error)
	CreateAccount(budgetID string, payload CreateAccountPayload) (*CreateAccountResponse, error)
}

type Syncer struct {
	store     *SyncStore
	client    YNABClient
	mapper    *Mapper
	budgetID  string
	startDate time.Time
}

type SyncResult struct {
	Total   int
	Synced  int
	Skipped int
	Failed  []string
}

func NewSyncer(store *SyncStore, client YNABClient, mapper *Mapper, budgetID string, startDate time.Time) *Syncer {
	return &Syncer{
		store:     store,
		client:    client,
		mapper:    mapper,
		budgetID:  budgetID,
		startDate: startDate,
	}
}

func (s *Syncer) Sync(messages []*message.Message, transactions []*template.Transaction) (*SyncResult, error) {
	result := &SyncResult{
		Total: len(transactions),
	}

	if len(messages) != len(transactions) {
		return nil, fmt.Errorf("messages and transactions length mismatch: %d vs %d", len(messages), len(transactions))
	}

	var toSync []TransactionPayload
	var toSyncImportIDs []string

	for i := 0; i < len(transactions); i++ {
		msg := messages[i]
		tx := transactions[i]

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

		toSync = append(toSync, *payload)
		toSyncImportIDs = append(toSyncImportIDs, importID)
	}

	if len(toSync) == 0 {
		return result, nil
	}

	// YNAB API limit: 100 transactions per request
	batchSize := 100
	for i := 0; i < len(toSync); i += batchSize {
		end := i + batchSize
		if end > len(toSync) {
			end = len(toSync)
		}

		batch := toSync[i:end]
		batchImportIDs := toSyncImportIDs[i:end]

		_, err := s.client.CreateTransactions(s.budgetID, batch)
		if err != nil {
			return result, fmt.Errorf("failed to create transactions: %w", err)
		}

		for _, importID := range batchImportIDs {
			record := &SyncRecord{
				ImportID: importID,
				SyncedAt: time.Now().UTC(),
			}
			if err := s.store.RecordSync(record); err != nil {
				return result, fmt.Errorf("failed to record sync: %w", err)
			}
			result.Synced++
		}
	}

	return result, nil
}
