# Transaction Analyzer Stage

## Overview

Extract transfer detection out of `ynab/syncer.go` into a dedicated `analyzer` package and fix three bugs:
1. "Tranzactia din..." (EximTransactionTemplate) messages pass `filterForSync` but fail in `MatchAccount` (no Card field) — causing noise in `result.Failed`
2. `detectTransferPairs` uses a 5-minute window — too narrow for cross-bank transfers (Exim→MAIB can take days)
3. Transfer detection logic is buried inside the sync step instead of being a separate, testable stage

The result is an explicit 3-stage pipeline: **parse → analyze → sync**.

## Context (from discovery)

- `main.go` — `filterForSync` (line 295), `runYNABSyncWithOptions` (line 316)
- `ynab/syncer.go` — `detectTransferPairs`, `Sync()` method
- `ynab/syncer_test.go` — existing tests for `Sync()`
- `template/template.go` — `EximTransactionTemplate`, `Transaction` struct, `Matcher`
- `message/message.go` — `Message` struct with `Sender` field
- NEW: `analyzer/analyzer.go`, `analyzer/analyzer_test.go`

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- All tests must pass before starting the next task
- Run `go test ./...` after each task

## Technical Details

### Types

```go
// analyzer/analyzer.go
type TransactionKind int

const (
    KindPayment  TransactionKind = iota // debit, no paired credit
    KindIncome                          // credit, no paired debit
    KindTransfer                        // debit side of a matched pair
    KindCreditTransfer                  // credit side of a matched pair — skip in sync
)

type AnalyzedTransaction struct {
    Message     *message.Message
    Transaction *template.Transaction
    Kind        TransactionKind
    PairIndex   int  // index of the paired transaction; -1 if none (Kind != KindTransfer/KindCreditTransfer)
    HasPair     bool // true when PairIndex is meaningful; always check this before using PairIndex
}

func Analyze(
    messages []*message.Message,
    transactions []*template.Transaction,
    sameBankWindow time.Duration,
    crossBankWindow time.Duration,
) []AnalyzedTransaction
```

Note: `PairIndex` is only valid when `HasPair == true`. Always guard with `if at.HasPair` before accessing `analyzed[at.PairIndex]`.

### Pairing logic

1. First pass: classify each transaction as debit (`isDebit(op) == true`) or credit (everything else)
2. For each debit at index `i`, search for the best credit match at index `j`:
   - Amount difference ≤ 0.01 (in converted currency)
   - `extractLast4(tx[i].Card) != extractLast4(tx[j].Card)` (different card — prevents self-pairing)
   - Time difference within window, determined by comparing the two message senders:
     - `messages[i].Sender == messages[j].Sender` → same-bank window (default 5 min)
     - `messages[i].Sender != messages[j].Sender` → cross-bank window (default 7 days)
   - First unmatched candidate wins (greedy), mark credit as used
3. Matched debit → `KindTransfer`, matched credit → `KindCreditTransfer`, both get `HasPair = true`, `PairIndex` pointing to each other
4. Unmatched debit → `KindPayment`, unmatched credit → `KindIncome` (both with `HasPair = false`)

`extractLast4` lives in the `analyzer` package (moved from `ynab/syncer.go`).

### Updated `Sync()` signature

```go
// ynab/syncer.go
func (s *Syncer) Sync(analyzed []analyzer.AnalyzedTransaction) (*SyncResult, error)
```

Credit-transfer side is detected by `at.Kind == analyzer.KindCreditTransfer` and skipped.
Transfer account ID is looked up via `analyzed[at.PairIndex].Transaction`.

### Updated pipeline in `main.go`

```
messages → parse → convertTransactions → filterForSync → analyzer.Analyze() → syncer.Sync()
```

`filterForSync` gains a `ShouldIgnore` check to exclude EximTransaction messages before they reach the mapper.

## Implementation Steps

### Task 1: Fix filterForSync — exclude ShouldIgnore messages

**Files:**
- Modify: `main.go`

Note: `EximTransactionTemplate` matches "Tranzactia din..." messages (they have `HasTemplate = true`), but they lack a `Card` field so `MatchAccount` fails. The `ShouldIgnore` check already exists for `runMissingTemplates` — we need to apply it in the sync path too.

- [ ] In `filterForSync`, add `app.matcher.ShouldIgnore(pm.Message.Content)` check; if true, increment `Skipped` and continue (do not add to filtered list)
- [ ] Write test in `main_test.go` (or integration test): create a `ParsedMessage` with EximTransaction-format content (`HasTemplate = true`), call `filterForSync`, verify it is excluded from the result
- [ ] Run `go test ./...` — must pass

### Task 2: Create `analyzer` package

**Files:**
- Create: `analyzer/analyzer.go`
- Create: `analyzer/analyzer_test.go`

- [ ] Define `TransactionKind` constants and `AnalyzedTransaction` struct (with `HasPair bool` + `PairIndex int`)
- [ ] Move `extractLast4` helper from `ynab/syncer.go` into `analyzer/analyzer.go` (it logically belongs to the pairing logic)
- [ ] Implement `Analyze()` with two-pass logic (classify → pair)
- [ ] Write tests:
  - MAIB→MAIB transfer: same sender, amounts match, different cards, within 5 min → both get KindTransfer/KindCreditTransfer
  - MAIB→MAIB: same card → NOT paired (remains KindPayment + KindIncome)
  - EXIM→MAIB: different senders, amounts match, 2 days apart → paired with cross-bank window
  - EXIM→MAIB: different senders, 8 days apart → NOT paired (exceeds cross-bank window)
  - Unmatched debit → KindPayment; unmatched credit → KindIncome
  - Multiple transactions, only one pair matches
- [ ] Run `go test ./analyzer/...` — must pass

### Task 3: Update `syncer.go` — remove `detectTransferPairs`, update `Sync()`

**Files:**
- Modify: `ynab/syncer.go`
- Modify: `ynab/syncer_test.go`

- [ ] Delete `detectTransferPairs` function from `syncer.go`
- [ ] Update `Sync()` to accept `[]analyzer.AnalyzedTransaction` instead of `([]*message.Message, []*template.Transaction)`
- [ ] Inside `Sync()`: skip `KindCreditTransfer` entries; for `KindTransfer` entries (where `at.HasPair == true`), set `payload.TransferAccountID` via `mapper.MatchAccount(analyzed[at.PairIndex].Transaction)`
- [ ] Remove the `creditSideIndexes` / `creditToDebit` maps — no longer needed
- [ ] Update `syncer_test.go`:
  - Replace direct `messages + transactions` inputs with `[]AnalyzedTransaction`
  - Add test for transfer case (KindTransfer sets TransferAccountID, KindCreditTransfer is skipped)
  - Add test for income (KindIncome is synced as a positive transaction)
- [ ] Run `go test ./ynab/...` — must pass

### Task 4: Wire analyzer into `main.go`

**Files:**
- Modify: `main.go`

- [ ] After `filterForSync`, call `analyzer.Analyze(filteredMessages, filteredTransactions, 5*time.Minute, 7*24*time.Hour)`
- [ ] Pass the result to `syncer.Sync(analyzed)`
- [ ] Remove the `filteredMessages` / `filteredTransactions` variables that are now consumed by the analyzer call
- [ ] Run `go test ./...` — must pass

### Task 5: Verify acceptance criteria

- [ ] Transfer MAIB→MAIB: synced as a single transfer with `TransferAccountID` set, credit side skipped
- [ ] Transfer EXIM→MAIB: same outcome with cross-bank window
- [ ] Regular payment: synced as `KindPayment`, no `TransferAccountID`
- [ ] "Tranzactia din..." messages: never appear in `result.Failed`
- [ ] Run full test suite: `go test ./...`
- [ ] Run `go vet ./...`

### Task 6: Update documentation and move plan

- [ ] Update `README.md` if it describes the pipeline
- [ ] Move this plan to `docs/plans/completed/`

## Post-Completion

- Manual smoke test: run `ynab_sync` against real chat.db and verify no unexpected duplicates or failures in sync output
- Consider adjusting the cross-bank window (7 days) if real transfers take longer
