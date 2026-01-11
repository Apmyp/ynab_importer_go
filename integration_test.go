//go:build integration
// +build integration

package main

import (
	"os"
	"testing"
)

// TestIntegration_Run runs the actual program with real database
// Run with: go test -tags=integration ./...
func TestIntegration_Run(t *testing.T) {
	// Skip if database not available
	if os.Getenv("RUN_INTEGRATION") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION=true to run")
	}

	err := Run([]string{})
	if err != nil {
		t.Errorf("Run() error: %v", err)
	}
}

func TestIntegration_MissingTemplates(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION=true to run")
	}

	err := Run([]string{"missing_templates"})
	if err != nil {
		t.Errorf("Run(missing_templates) error: %v", err)
	}
}
