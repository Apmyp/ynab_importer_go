package main

import (
	"fmt"

	"github.com/apmyp/ynab_importer_go/formatter"
	"github.com/apmyp/ynab_importer_go/parser"
	"github.com/apmyp/ynab_importer_go/reader"
	"github.com/apmyp/ynab_importer_go/ynab"
)

// App encapsulates the application logic
type App struct {
	reader    *reader.Reader
	parser    *parser.Parser
	formatter *formatter.Formatter
	client    *ynab.Client
}

// NewApp creates a new application instance
func NewApp() *App {
	return &App{
		reader:    reader.NewReader(),
		parser:    parser.NewParser(),
		formatter: formatter.NewFormatter(),
		client:    ynab.NewClient(),
	}
}

// Run is the main application entry point
func Run() error {
	fmt.Println("YNAB Importer")
	app := NewApp()
	_ = app
	return nil
}

func main() {
	if err := Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
