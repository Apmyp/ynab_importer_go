package analyzer

import (
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/apmyp/ynab_importer_go/message"
	"github.com/apmyp/ynab_importer_go/template"
)

type TransactionKind int

const (
	KindPayment        TransactionKind = iota // debit, no paired credit
	KindIncome                                // credit, no paired debit
	KindTransfer                              // debit side of a matched pair
	KindCreditTransfer                        // credit side of a matched pair — skip in sync
)

type AnalyzedTransaction struct {
	Message     *message.Message
	Transaction *template.Transaction
	Kind        TransactionKind
	PairIndex   int  // index of the paired transaction; -1 if none
	HasPair     bool // true when PairIndex is meaningful
}

var last4Regex = regexp.MustCompile(`\d{4}$`)

func extractLast4(card string) string {
	return last4Regex.FindString(card)
}

func isDebit(operation string) bool {
	debitOps := []string{
		"Debitare",
		"Tovary i uslugi",
		"Tranzactie reusita",
		"Nalog na doxody po vkladu",
	}
	for _, op := range debitOps {
		if strings.Contains(operation, op) {
			return true
		}
	}
	return false
}

// Analyze classifies transactions and detects transfer pairs.
// sameBankWindow is used when both messages share the same Sender.
// crossBankWindow is used for messages from different senders.
func Analyze(
	messages []*message.Message,
	transactions []*template.Transaction,
	sameBankWindow time.Duration,
	crossBankWindow time.Duration,
) []AnalyzedTransaction {
	n := len(transactions)
	analyzed := make([]AnalyzedTransaction, n)
	for i := range analyzed {
		analyzed[i] = AnalyzedTransaction{
			Message:     messages[i],
			Transaction: transactions[i],
			PairIndex:   -1,
		}
	}

	creditUsed := make([]bool, n)

	// For each debit, find the best unmatched credit.
	for i := 0; i < n; i++ {
		if !isDebit(transactions[i].Operation) {
			continue
		}
		for j := 0; j < n; j++ {
			if i == j || isDebit(transactions[j].Operation) || creditUsed[j] {
				continue
			}
			if math.Abs(transactions[i].Converted.Value-transactions[j].Converted.Value) > 0.01 {
				continue
			}
			if extractLast4(transactions[i].Card) == extractLast4(transactions[j].Card) {
				continue
			}

			var window time.Duration
			if messages[i].Sender == messages[j].Sender {
				window = sameBankWindow
			} else {
				window = crossBankWindow
			}

			timeDiff := messages[i].Timestamp.Sub(messages[j].Timestamp)
			if timeDiff < 0 {
				timeDiff = -timeDiff
			}
			if timeDiff > window {
				continue
			}

			analyzed[i].Kind = KindTransfer
			analyzed[i].HasPair = true
			analyzed[i].PairIndex = j

			analyzed[j].Kind = KindCreditTransfer
			analyzed[j].HasPair = true
			analyzed[j].PairIndex = i

			creditUsed[j] = true
			break
		}
	}

	// Classify unmatched entries.
	for i := range analyzed {
		if !analyzed[i].HasPair {
			if isDebit(transactions[i].Operation) {
				analyzed[i].Kind = KindPayment
			} else {
				analyzed[i].Kind = KindIncome
			}
		}
	}

	return analyzed
}
