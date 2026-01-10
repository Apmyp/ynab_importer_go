package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Error("NewApp() should return a non-nil App")
	}
	if app.reader == nil {
		t.Error("App.reader should not be nil")
	}
	if app.parser == nil {
		t.Error("App.parser should not be nil")
	}
	if app.formatter == nil {
		t.Error("App.formatter should not be nil")
	}
	if app.client == nil {
		t.Error("App.client should not be nil")
	}
}

func TestRun(t *testing.T) {
	// Test that Run function executes without error
	err := Run()
	if err != nil {
		t.Errorf("Run() returned error: %v", err)
	}
}

func TestRunOutput(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("Run() returned error: %v", err)
	}

	if output != "YNAB Importer\n" {
		t.Errorf("Run() output = %q, want %q", output, "YNAB Importer\n")
	}
}

func TestMainFunction(t *testing.T) {
	// We test main indirectly by testing the full flow through Run
	// This ensures the initialization path is covered
	app := NewApp()
	if app.reader == nil || app.parser == nil || app.formatter == nil || app.client == nil {
		t.Error("App components should be properly initialized")
	}

	// Verify Run works in the context main would call it
	if err := Run(); err != nil {
		t.Errorf("Run should not return error in normal flow: %v", err)
	}
}
