package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/apmyp/ynab_importer_go/bagoup"
	"github.com/apmyp/ynab_importer_go/chatdb"
	"github.com/apmyp/ynab_importer_go/config"
	"github.com/apmyp/ynab_importer_go/exchangerate"
	"github.com/apmyp/ynab_importer_go/system"
	"github.com/apmyp/ynab_importer_go/template"
	"github.com/apmyp/ynab_importer_go/worker"
	"github.com/apmyp/ynab_importer_go/ynab"
)

// MessageFetcher defines the interface for fetching messages
type MessageFetcher interface {
	FetchMessages() ([]*bagoup.Message, func(), error)
	CheckDependencies() error
}

// ChatDBFetcher implements MessageFetcher using direct SQLite access
type ChatDBFetcher struct {
	config *config.Config
}

// NewChatDBFetcher creates a new ChatDBFetcher
func NewChatDBFetcher(cfg *config.Config) *ChatDBFetcher {
	return &ChatDBFetcher{
		config: cfg,
	}
}

// expandPath expands ~ to home directory in file paths
func expandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if path == "~" {
		return homeDir, nil
	}

	return filepath.Join(homeDir, path[2:]), nil
}

// CheckDependencies verifies chat.db is accessible
func (f *ChatDBFetcher) CheckDependencies() error {
	dbPath, err := expandPath(f.config.DBPath)
	if err != nil {
		return err
	}

	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("chat.db not accessible at %s: %w", dbPath, err)
	}
	return nil
}

// FetchMessages reads messages directly from chat.db
func (f *ChatDBFetcher) FetchMessages() ([]*bagoup.Message, func(), error) {
	dbPath, err := expandPath(f.config.DBPath)
	if err != nil {
		return nil, func() {}, err
	}

	reader, err := chatdb.NewReader(dbPath, f.config.Senders)
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to open chat.db: %w", err)
	}

	messages, err := reader.FetchMessages()
	if err != nil {
		reader.Close()
		return nil, func() {}, fmt.Errorf("failed to fetch messages: %w", err)
	}

	cleanup := func() {
		if err := reader.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close chat.db: %v\n", err)
		}
	}

	fmt.Printf("Loaded %d messages from chat.db\n", len(messages))
	return messages, cleanup, nil
}

// App encapsulates the application logic
type App struct {
	config     *config.Config
	configPath string
	fetcher    MessageFetcher
	matcher    *template.Matcher
	pool       *worker.Pool
	converter  *exchangerate.Converter
}

// createExchangeRateStore creates an exchange rate store with warning on failure
func createExchangeRateStore(dataFilePath string) *exchangerate.Store {
	store, err := exchangerate.NewStore(dataFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize exchange rate store: %v\n", err)
		return nil
	}
	return store
}

// NewApp creates a new application instance
func NewApp(cfg *config.Config, configPath string) *App {
	return &App{
		config:     cfg,
		configPath: configPath,
		fetcher:    NewChatDBFetcher(cfg),
		matcher:    template.NewMatcher(),
		pool:       worker.NewPool(runtime.NumCPU()),
		converter:  exchangerate.NewConverter(createExchangeRateStore(cfg.DataFilePath), exchangerate.NewFetcher(), cfg.DefaultCurrency),
	}
}

// NewAppWithFetcher creates a new application with a custom fetcher (for testing)
func NewAppWithFetcher(cfg *config.Config, fetcher MessageFetcher) *App {
	return &App{
		config:    cfg,
		fetcher:   fetcher,
		matcher:   template.NewMatcher(),
		pool:      worker.NewPool(runtime.NumCPU()),
		converter: exchangerate.NewConverter(createExchangeRateStore(cfg.DataFilePath), exchangerate.NewFetcher(), cfg.DefaultCurrency),
	}
}

// ParsedMessage holds a message and its parsed transaction (if any)
type ParsedMessage struct {
	Message     *bagoup.Message
	Transaction *template.Transaction
	HasTemplate bool
}

// Run is the main application entry point
func Run(args []string) error {
	configPath := "config.json"
	dataFilePath := ""

	// Parse options
	for len(args) > 0 {
		if args[0] == "--config" && len(args) > 1 {
			configPath = args[1]
			args = args[2:]
		} else if args[0] == "--data-file" && len(args) > 1 {
			dataFilePath = args[1]
			args = args[2:]
		} else {
			break
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Override data file path if specified via command line
	if dataFilePath != "" {
		cfg.DataFilePath = dataFilePath
	}

	app := NewApp(cfg, configPath)

	// Check dependencies
	if err := app.fetcher.CheckDependencies(); err != nil {
		return err
	}

	// Determine command (ynab_sync is default)
	command := "ynab_sync"
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "missing_templates":
		return app.runMissingTemplates()
	case "ynab_sync":
		return app.runYNABSync()
	case "system_install":
		return app.runSystemInstall()
	case "system_uninstall":
		return app.runSystemUninstall()
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// runMissingTemplates outputs messages without matching templates (excluding ignored messages)
func (app *App) runMissingTemplates() error {
	messages, cleanup, err := app.fetchMessages()
	if err != nil {
		return err
	}
	defer cleanup()

	fmt.Println("Messages without matching templates:")
	fmt.Println("=====================================")

	// Check templates and ignore patterns in parallel
	type checkResult struct {
		hasTemplate  bool
		shouldIgnore bool
	}
	results := make([]checkResult, len(messages))
	app.pool.Map(len(messages), func(i int) {
		content := messages[i].Content
		results[i] = checkResult{
			hasTemplate:  app.matcher.FindTemplate(content) != nil,
			shouldIgnore: app.matcher.ShouldIgnore(content),
		}
	})

	count := 0
	for i, msg := range messages {
		// Skip user's own messages (sender "Me" from bagoup)
		if msg.Sender == "Me" {
			continue
		}
		// Skip messages that have a template or should be ignored
		if results[i].hasTemplate || results[i].shouldIgnore {
			continue
		}
		count++
		fmt.Printf("\n[%s] %s: [%d chars]\n",
			msg.Timestamp.Format("2006-01-02 15:04:05"),
			msg.Sender,
			len(msg.Content))
		fmt.Println("---")
	}

	fmt.Printf("\nTotal messages without templates: %d\n", count)
	return nil
}

// fetchMessages uses the fetcher to get messages
func (app *App) fetchMessages() ([]*bagoup.Message, func(), error) {
	return app.fetcher.FetchMessages()
}

// parseMessage attempts to parse a message using templates
func (app *App) parseMessage(msg *bagoup.Message) *ParsedMessage {
	tx, err := app.matcher.Parse(msg.Content)
	return &ParsedMessage{
		Message:     msg,
		Transaction: tx,
		HasTemplate: err == nil,
	}
}

// validateYNABConfig validates YNAB configuration
func (app *App) validateYNABConfig() error {
	if app.config.YNAB.BudgetID == "" {
		return fmt.Errorf("YNAB budget_id not configured")
	}
	if app.config.YNAB.StartDate == "" {
		return fmt.Errorf("YNAB start_date not configured")
	}
	return nil
}

// fetchFirstBudgetID fetches the first budget ID from YNAB API
func (app *App) fetchFirstBudgetID(client *ynab.HTTPClient) (string, error) {
	resp, err := client.GetBudgets()
	if err != nil {
		return "", err
	}
	if len(resp.Data.Budgets) == 0 {
		return "", fmt.Errorf("no budgets found in YNAB account")
	}
	return resp.Data.Budgets[0].ID, nil
}

// convertTransactions converts all foreign currency transactions to default currency
func (app *App) convertTransactions(parsedMessages []*ParsedMessage) {
	if app.converter == nil {
		return
	}

	for _, pm := range parsedMessages {
		if pm == nil || !pm.HasTemplate || pm.Transaction == nil {
			continue
		}

		tx := pm.Transaction
		date := pm.Message.Timestamp.UTC().Truncate(24 * time.Hour)

		rate, err := app.converter.GetOrFetchRate(date, tx.Original.Currency)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get exchange rate for %s on %s: %v\n",
				tx.Original.Currency, date.Format("2006-01-02"), err)
			tx.Converted = tx.Original
			continue
		}

		tx.Converted = template.Amount{
			Value:    tx.Original.Value * rate,
			Currency: app.config.DefaultCurrency,
		}
	}
}

// runYNABSync synchronizes transactions to YNAB
func (app *App) runYNABSync() error {
	// Check API key first (needed for budget fetch)
	apiKey := os.Getenv("YNAB_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("YNAB_API_KEY environment variable not set")
	}

	// Fetch budget ID from API if not configured
	if app.config.YNAB.BudgetID == "" {
		client := ynab.NewHTTPClient(apiKey)
		budgetID, err := app.fetchFirstBudgetID(client)
		client.ClearAPIKey()
		if err != nil {
			return fmt.Errorf("failed to fetch budget ID: %w", err)
		}
		app.config.YNAB.BudgetID = budgetID
		if err := app.config.Save(app.configPath); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Printf("Saved budget ID %s to config\n", budgetID)
	}

	// Validate remaining YNAB configuration
	if err := app.validateYNABConfig(); err != nil {
		return err
	}

	// Parse start date
	startDate, err := time.Parse("2006-01-02", app.config.YNAB.StartDate)
	if err != nil {
		return fmt.Errorf("invalid YNAB start_date format: %w", err)
	}

	// Fetch messages
	messages, cleanup, err := app.fetchMessages()
	if err != nil {
		return err
	}
	defer cleanup()

	// Parse messages in parallel
	parsedMessages := make([]*ParsedMessage, len(messages))
	var mu sync.Mutex

	app.pool.Map(len(messages), func(i int) {
		parsed := app.parseMessage(messages[i])
		mu.Lock()
		parsedMessages[i] = parsed
		mu.Unlock()
	})

	// Convert foreign currencies to MDL
	app.convertTransactions(parsedMessages)

	// Filter to only include transactions with templates
	var filteredMessages []*bagoup.Message
	var filteredTransactions []*template.Transaction
	for _, pm := range parsedMessages {
		if pm != nil && pm.HasTemplate && pm.Transaction != nil {
			// Skip declined transactions
			if strings.HasPrefix(pm.Transaction.Status, "Decline") {
				continue
			}

			// Only include MDL transactions
			if pm.Transaction.Converted.Currency == "MDL" {
				filteredMessages = append(filteredMessages, pm.Message)
				filteredTransactions = append(filteredTransactions, pm.Transaction)
			}
		}
	}

	fmt.Printf("Found %d MDL transactions to sync\n", len(filteredTransactions))

	// Initialize YNAB components
	syncStore, err := ynab.NewSyncStore(app.config.DataFilePath)
	if err != nil {
		return fmt.Errorf("failed to initialize sync store: %w", err)
	}
	defer syncStore.Close()

	client := ynab.NewHTTPClient(apiKey)
	defer client.ClearAPIKey()

	// Ensure all transactions have YNAB accounts
	accountManager := ynab.NewAccountManager(client)
	updatedAccounts, err := accountManager.EnsureAccounts(
		app.config.YNAB.BudgetID,
		app.config.YNAB.Accounts,
		filteredTransactions,
	)
	if err != nil {
		return fmt.Errorf("failed to ensure accounts: %w", err)
	}

	// Save config if new accounts were added
	if len(updatedAccounts) > len(app.config.YNAB.Accounts) {
		numNewAccounts := len(updatedAccounts) - len(app.config.YNAB.Accounts)
		app.config.YNAB.Accounts = updatedAccounts
		if err := app.config.Save(app.configPath); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Printf("Added %d new account(s) to config\n", numNewAccounts)
	}

	// Convert config accounts to ynab accounts
	ynabAccounts := make([]ynab.YNABAccount, len(updatedAccounts))
	for i, acc := range updatedAccounts {
		ynabAccounts[i] = ynab.YNABAccount{
			YNABAccountID: acc.YNABAccountID,
			Last4:         acc.Last4,
		}
	}

	mapper := ynab.NewMapper(ynabAccounts)
	syncer := ynab.NewSyncer(syncStore, client, mapper, app.config.YNAB.BudgetID, startDate)

	// Sync transactions
	result, err := syncer.Sync(filteredMessages, filteredTransactions)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Display results
	fmt.Printf("\nSync Results:\n")
	fmt.Printf("  Total transactions: %d\n", result.Total)
	fmt.Printf("  Synced: %d\n", result.Synced)
	fmt.Printf("  Skipped: %d\n", result.Skipped)
	if len(result.Failed) > 0 {
		fmt.Printf("  Failed: %d\n", len(result.Failed))
		for _, failure := range result.Failed {
			fmt.Printf("    - %s\n", failure)
		}
	}

	return nil
}

// runSystemInstall installs the hourly sync service
func (app *App) runSystemInstall() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	installer, err := system.NewInstaller(execPath, workingDir)
	if err != nil {
		return err
	}

	if err := installer.Install(); err != nil {
		return err
	}

	fmt.Println("Successfully installed hourly sync service")
	fmt.Printf("Binary: %s\n", execPath)
	fmt.Printf("Working directory: %s\n", workingDir)
	fmt.Printf("Logs:\n")
	fmt.Printf("  Standard output: %s/ynab_sync.log\n", workingDir)
	fmt.Printf("  Error output: %s/ynab_sync_error.log\n", workingDir)
	fmt.Println("Sync will run every hour")

	return nil
}

// runSystemUninstall removes the hourly sync service
func (app *App) runSystemUninstall() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	installer, err := system.NewInstaller(execPath, workingDir)
	if err != nil {
		return err
	}

	if err := installer.Uninstall(); err != nil {
		return err
	}

	fmt.Println("Successfully uninstalled hourly sync service")
	return nil
}

func main() {
	if err := Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
