package analyzer

import (
	"testing"
	"time"

	"github.com/apmyp/ynab_importer_go/message"
	"github.com/apmyp/ynab_importer_go/template"
)

func TestAnalyze_MAIBtoMAIB_SameSender_DetectsPair(t *testing.T) {
	base := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	msgs := []*message.Message{
		{Timestamp: base, Sender: "102"},
		{Timestamp: base.Add(2 * time.Minute), Sender: "102"},
	}
	txs := []*template.Transaction{
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..2222", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Suplinire"},
	}

	result := Analyze(msgs, txs, 5*time.Minute, 7*24*time.Hour)

	if result[0].Kind != KindTransfer {
		t.Errorf("result[0].Kind = %v, want KindTransfer", result[0].Kind)
	}
	if result[1].Kind != KindCreditTransfer {
		t.Errorf("result[1].Kind = %v, want KindCreditTransfer", result[1].Kind)
	}
	if !result[0].HasPair || result[0].PairIndex != 1 {
		t.Errorf("result[0].HasPair=%v PairIndex=%d, want true/1", result[0].HasPair, result[0].PairIndex)
	}
	if !result[1].HasPair || result[1].PairIndex != 0 {
		t.Errorf("result[1].HasPair=%v PairIndex=%d, want true/0", result[1].HasPair, result[1].PairIndex)
	}
}

func TestAnalyze_MAIBtoMAIB_SameCard_NotPaired(t *testing.T) {
	base := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	msgs := []*message.Message{
		{Timestamp: base, Sender: "102"},
		{Timestamp: base.Add(2 * time.Minute), Sender: "102"},
	}
	txs := []*template.Transaction{
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Suplinire"},
	}

	result := Analyze(msgs, txs, 5*time.Minute, 7*24*time.Hour)

	if result[0].Kind != KindPayment {
		t.Errorf("result[0].Kind = %v, want KindPayment (same card, not paired)", result[0].Kind)
	}
	if result[1].Kind != KindIncome {
		t.Errorf("result[1].Kind = %v, want KindIncome (same card, not paired)", result[1].Kind)
	}
}

func TestAnalyze_EXIMtoMAIB_CrossBank_DetectsPair(t *testing.T) {
	base := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	msgs := []*message.Message{
		{Timestamp: base, Sender: "EXIMBANK"},
		{Timestamp: base.Add(2 * 24 * time.Hour), Sender: "MAIB"},
	}
	txs := []*template.Transaction{
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..2222", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Suplinire"},
	}

	result := Analyze(msgs, txs, 5*time.Minute, 7*24*time.Hour)

	if result[0].Kind != KindTransfer {
		t.Errorf("result[0].Kind = %v, want KindTransfer (cross-bank, 2 days)", result[0].Kind)
	}
	if result[1].Kind != KindCreditTransfer {
		t.Errorf("result[1].Kind = %v, want KindCreditTransfer", result[1].Kind)
	}
}

func TestAnalyze_EXIMtoMAIB_ExceedsCrossBankWindow_NotPaired(t *testing.T) {
	base := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	msgs := []*message.Message{
		{Timestamp: base, Sender: "EXIMBANK"},
		{Timestamp: base.Add(8 * 24 * time.Hour), Sender: "MAIB"},
	}
	txs := []*template.Transaction{
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..2222", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Suplinire"},
	}

	result := Analyze(msgs, txs, 5*time.Minute, 7*24*time.Hour)

	if result[0].Kind != KindPayment {
		t.Errorf("result[0].Kind = %v, want KindPayment (8 days exceeds cross-bank window)", result[0].Kind)
	}
	if result[1].Kind != KindIncome {
		t.Errorf("result[1].Kind = %v, want KindIncome", result[1].Kind)
	}
}

func TestAnalyze_UnmatchedDebit_KindPayment(t *testing.T) {
	msgs := []*message.Message{
		{Timestamp: time.Now(), Sender: "102"},
	}
	txs := []*template.Transaction{
		{Card: "9..1234", Converted: template.Amount{Value: 100.00, Currency: "MDL"}, Operation: "Debitare"},
	}

	result := Analyze(msgs, txs, 5*time.Minute, 7*24*time.Hour)

	if result[0].Kind != KindPayment {
		t.Errorf("result[0].Kind = %v, want KindPayment", result[0].Kind)
	}
	if result[0].HasPair {
		t.Error("result[0].HasPair = true, want false for unmatched debit")
	}
}

func TestAnalyze_UnmatchedCredit_KindIncome(t *testing.T) {
	msgs := []*message.Message{
		{Timestamp: time.Now(), Sender: "102"},
	}
	txs := []*template.Transaction{
		{Card: "9..1234", Converted: template.Amount{Value: 93719.33, Currency: "MDL"}, Operation: "Suplinire"},
	}

	result := Analyze(msgs, txs, 5*time.Minute, 7*24*time.Hour)

	if result[0].Kind != KindIncome {
		t.Errorf("result[0].Kind = %v, want KindIncome", result[0].Kind)
	}
	if result[0].HasPair {
		t.Error("result[0].HasPair = true, want false for unmatched credit")
	}
}

func TestAnalyze_KommunalnyePlatezhi_KindPayment(t *testing.T) {
	msgs := []*message.Message{
		{Timestamp: time.Now(), Sender: "102"},
	}
	txs := []*template.Transaction{
		{Card: "*1972", Converted: template.Amount{Value: 885.26, Currency: "MDL"}, Operation: "Kommunal'nye platezhi"},
	}

	result := Analyze(msgs, txs, 5*time.Minute, 7*24*time.Hour)

	if result[0].Kind != KindPayment {
		t.Errorf("result[0].Kind = %v, want KindPayment for Kommunal'nye platezhi", result[0].Kind)
	}
}

func TestAnalyze_MAIBtoMAIB_RaznyeVyplaty_DetectsPair(t *testing.T) {
	base := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	msgs := []*message.Message{
		{Timestamp: base, Sender: "102"},
		{Timestamp: base.Add(1 * time.Minute), Sender: "102"},
	}
	txs := []*template.Transaction{
		{Card: "*1972", Converted: template.Amount{Value: 50000.00, Currency: "MDL"}, Operation: "Raznye vyplaty *4335"},
		{Card: "*4335", Converted: template.Amount{Value: 50000.00, Currency: "MDL"}, Operation: "Popolnenie *1972"},
	}

	result := Analyze(msgs, txs, 5*time.Minute, 7*24*time.Hour)

	if result[0].Kind != KindTransfer {
		t.Errorf("result[0].Kind = %v, want KindTransfer", result[0].Kind)
	}
	if result[1].Kind != KindCreditTransfer {
		t.Errorf("result[1].Kind = %v, want KindCreditTransfer", result[1].Kind)
	}
	if !result[0].HasPair || result[0].PairIndex != 1 {
		t.Errorf("result[0].HasPair=%v PairIndex=%d, want true/1", result[0].HasPair, result[0].PairIndex)
	}
}

func TestAnalyze_CreditUsedOnlyOnce(t *testing.T) {
	base := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	msgs := []*message.Message{
		{Timestamp: base, Sender: "102"},
		{Timestamp: base.Add(1 * time.Minute), Sender: "102"},
		{Timestamp: base.Add(2 * time.Minute), Sender: "102"},
	}
	txs := []*template.Transaction{
		{Card: "9..1111", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..3333", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Debitare"},
		{Card: "9..2222", Converted: template.Amount{Value: 500.00, Currency: "MDL"}, Operation: "Suplinire"},
	}

	result := Analyze(msgs, txs, 5*time.Minute, 7*24*time.Hour)

	// First debit should claim the credit.
	if result[0].Kind != KindTransfer {
		t.Errorf("result[0].Kind = %v, want KindTransfer", result[0].Kind)
	}
	if result[2].Kind != KindCreditTransfer {
		t.Errorf("result[2].Kind = %v, want KindCreditTransfer", result[2].Kind)
	}
	// Second debit has no credit to pair with.
	if result[1].Kind != KindPayment {
		t.Errorf("result[1].Kind = %v, want KindPayment (credit already used)", result[1].Kind)
	}
}
