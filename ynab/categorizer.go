package ynab

import (
	"strings"
	"time"
)

const categorySeedTTL = 24 * time.Hour

func SeedCategoriesFromYNAB(client YNABClient, budgetID string, store *CategoryStore) error {
	fresh, err := store.IsSeededRecently(categorySeedTTL)
	if err != nil {
		return err
	}
	if fresh {
		return nil
	}

	sinceDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")

	resp, err := client.GetTransactions(budgetID, sinceDate)
	if err != nil {
		return err
	}

	mapping := make(map[string]string)
	for _, tx := range resp.Data.Transactions {
		if tx.PayeeName == "" || tx.CategoryID == "" {
			continue
		}
		if strings.Contains(tx.CategoryName, "Inflow") {
			continue
		}
		mapping[tx.PayeeName] = tx.CategoryID
	}

	if len(mapping) > 0 {
		if err := store.SetBatch(mapping); err != nil {
			return err
		}
	}

	return store.MarkSeeded()
}
