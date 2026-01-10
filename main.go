package main

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"

	"github.com/apmyp/ynab_importer_go/bagoup"
	"github.com/apmyp/ynab_importer_go/config"
	"github.com/apmyp/ynab_importer_go/template"
	"github.com/apmyp/ynab_importer_go/worker"
)

// MessageFetcher defines the interface for fetching messages
type MessageFetcher interface {
	FetchMessages() ([]*bagoup.Message, func(), error)
	CheckDependencies() error
}

// BagoupFetcher implements MessageFetcher using the bagoup command
type BagoupFetcher struct {
	runner *bagoup.Runner
	config *config.Config
}

// NewBagoupFetcher creates a new BagoupFetcher
func NewBagoupFetcher(cfg *config.Config) *BagoupFetcher {
	return &BagoupFetcher{
		runner: bagoup.NewRunner(),
		config: cfg,
	}
}

// CheckDependencies verifies bagoup is available
func (f *BagoupFetcher) CheckDependencies() error {
	return f.runner.CheckDependencies()
}

// FetchMessages runs bagoup and returns messages
func (f *BagoupFetcher) FetchMessages() ([]*bagoup.Message, func(), error) {
	f.runner.
		WithDBPath(f.config.Bagoup.DBPath).
		WithSenders(f.config.Senders)

	outputDir, err := f.runner.Run()
	if err != nil {
		return nil, func() {}, err
	}

	cleanup := func() {
		f.runner.Cleanup()
	}

	messages, err := f.runner.ReadAllMessages()
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}

	fmt.Printf("Loaded %d messages from %s\n", len(messages), outputDir)
	return messages, cleanup, nil
}

// App encapsulates the application logic
type App struct {
	config  *config.Config
	fetcher MessageFetcher
	matcher *template.Matcher
	pool    *worker.Pool
}

// NewApp creates a new application instance
func NewApp(cfg *config.Config) *App {
	numWorkers := runtime.NumCPU()
	return &App{
		config:  cfg,
		fetcher: NewBagoupFetcher(cfg),
		matcher: template.NewMatcher(),
		pool:    worker.NewPool(numWorkers),
	}
}

// NewAppWithFetcher creates a new application with a custom fetcher (for testing)
func NewAppWithFetcher(cfg *config.Config, fetcher MessageFetcher) *App {
	numWorkers := runtime.NumCPU()
	return &App{
		config:  cfg,
		fetcher: fetcher,
		matcher: template.NewMatcher(),
		pool:    worker.NewPool(numWorkers),
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
	if len(args) > 0 && args[0] == "--config" && len(args) > 1 {
		configPath = args[1]
		args = args[2:]
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	app := NewApp(cfg)

	// Check dependencies
	if err := app.fetcher.CheckDependencies(); err != nil {
		return err
	}

	// Determine command
	command := "default"
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "missing_templates":
		return app.runMissingTemplates()
	case "default":
		return app.runDefault()
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// runDefault fetches messages, parses them, and displays last 2 from each sender
func (app *App) runDefault() error {
	messages, cleanup, err := app.fetchMessages()
	if err != nil {
		return err
	}
	defer cleanup()

	// Parse messages in parallel using worker pool
	parsedMessages := make([]*ParsedMessage, len(messages))
	var mu sync.Mutex

	app.pool.Map(len(messages), func(i int) {
		parsed := app.parseMessage(messages[i])
		mu.Lock()
		parsedMessages[i] = parsed
		mu.Unlock()
	})

	// Group by sender
	bySender := make(map[string][]*ParsedMessage)
	for _, pm := range parsedMessages {
		if pm != nil {
			bySender[pm.Message.Sender] = append(bySender[pm.Message.Sender], pm)
		}
	}

	// Sort each sender's messages by timestamp (newest first) and take last 2
	for sender, msgs := range bySender {
		sort.Slice(msgs, func(i, j int) bool {
			return msgs[i].Message.Timestamp.After(msgs[j].Message.Timestamp)
		})

		fmt.Printf("\n=== %s (last 2 messages) ===\n", sender)
		count := 2
		if len(msgs) < 2 {
			count = len(msgs)
		}

		for i := 0; i < count; i++ {
			pm := msgs[i]
			fmt.Printf("\n[%s]\n", pm.Message.Timestamp.Format("2006-01-02 15:04:05"))
			if pm.HasTemplate && pm.Transaction != nil {
				fmt.Printf("Operation: %s\n", pm.Transaction.Operation)
				fmt.Printf("Amount: %.2f %s\n", pm.Transaction.Amount, pm.Transaction.Currency)
				fmt.Printf("Status: %s\n", pm.Transaction.Status)
				if pm.Transaction.Address != "" {
					fmt.Printf("Address: %s\n", pm.Transaction.Address)
				}
			} else {
				fmt.Printf("(no template)\n%s\n", pm.Message.Content)
			}
		}
	}

	return nil
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
		fmt.Printf("\n[%s] %s:\n%s\n",
			msg.Timestamp.Format("2006-01-02 15:04:05"),
			msg.Sender,
			msg.Content)
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

func main() {
	if err := Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
