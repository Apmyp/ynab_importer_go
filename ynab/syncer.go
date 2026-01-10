package ynab

import (
	"fmt"
	"time"

	"github.com/apmyp/ynab_importer_go/bagoup"
	"github.com/apmyp/ynab_importer_go/template"
)

// YNABClient defines the interface for YNAB API operations
type YNABClient interface {
	CreateTransactions(budgetID string, transactions []TransactionPayload) (*CreateTransactionsResponse, error)
}

// Syncer orchestrates syncing transactions to YNAB
type Syncer struct {
	store     *SyncStore
	client    YNABClient
	mapper    *Mapper
	budgetID  string
	startDate time.Time
}

// SyncResult contains the results of a sync operation
type SyncResult struct {
	Total   int
	Synced  int
	Skipped int
	Failed  []string
}

// NewSyncer creates a new Syncer
func NewSyncer(store *SyncStore, client YNABClient, mapper *Mapper, budgetID string, startDate time.Time) *Syncer {
	return &Syncer{
		store:     store,
		client:    client,
		mapper:    mapper,
		budgetID:  budgetID,
		startDate: startDate,
	}
}

// Sync synchronizes transactions to YNAB
func (s *Syncer) Sync(messages []*bagoup.Message, transactions []*template.Transaction) (*SyncResult, error) {
	result := &SyncResult{
		Total: len(transactions),
	}

	if len(messages) != len(transactions) {
		return nil, fmt.Errorf("messages and transactions length mismatch: %d vs %d", len(messages), len(transactions))
	}

	// Filter and map transactions
	var toSync []TransactionPayload
	var toSyncImportIDs []string

	for i := 0; i < len(transactions); i++ {
		msg := messages[i]
		tx := transactions[i]

		// Filter by start date
		if msg.Timestamp.Before(s.startDate) {
			result.Skipped++
			continue
		}

		// Generate import ID
		importID := s.mapper.GenerateImportID(msg, tx)

		// Check if already synced
		synced, err := s.store.IsSynced(importID)
		if err != nil {
			return nil, fmt.Errorf("failed to check sync status: %w", err)
		}
		if synced {
			result.Skipped++
			continue
		}

		// Map transaction
		payload, err := s.mapper.MapTransaction(msg, tx)
		if err != nil {
			// Skip transactions that can't be mapped (e.g., no account match)
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

	// Send in batches of 100 (YNAB API limit)
	batchSize := 100
	for i := 0; i < len(toSync); i += batchSize {
		end := i + batchSize
		if end > len(toSync) {
			end = len(toSync)
		}

		batch := toSync[i:end]
		batchImportIDs := toSyncImportIDs[i:end]

		// Send batch to YNAB
		_, err := s.client.CreateTransactions(s.budgetID, batch)
		if err != nil {
			return result, fmt.Errorf("failed to create transactions: %w", err)
		}

		// Record successful syncs
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
