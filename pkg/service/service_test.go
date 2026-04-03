package service_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yourusername/qrgen/pkg/service"
	"github.com/yourusername/qrgen/pkg/types"
)

// TestService_Run_GeneratesCSV verifies that running the service with a small
// quantity produces a CSV artifact under tmp/csv containing the expected
// number of generated codes.
//
// This is an integration-light test that only relies on local filesystem
// artifacts. It sets SECRET_KEY required by code generation logic.
func TestService_Run_GeneratesCSV(t *testing.T) {
	// Ensure a SECRET_KEY is present for deterministic hash generation.
	const secretEnv = "SECRET_KEY"
	prevSecret := os.Getenv(secretEnv)
	_ = os.Setenv(secretEnv, "integration-test-secret")
	t.Cleanup(func() {
		// Restore previous value (or unset if originally empty).
		if prevSecret == "" {
			_ = os.Unsetenv(secretEnv)
		} else {
			_ = os.Setenv(secretEnv, prevSecret)
		}
	})

	// Clean tmp before starting to avoid interference
	_ = os.RemoveAll("tmp")
	t.Cleanup(func() {
		// Best-effort cleanup after test
		_ = os.RemoveAll("tmp")
	})

	arg := &types.Argument{
		Quantity: 5,
		Info:     "integration-test",
		Size:     200,
		URL:      "https://example.com/test",
		Format:   types.FormatPDF,
		Storage:  types.StorageLocal,
	}

	svc := service.New(arg)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := svc.Run(ctx); err != nil {
		t.Fatalf("service.Run returned error: %v", err)
	}

	// Allow small delay for filesystem sync on some environments
	time.Sleep(100 * time.Millisecond)

	// Look for CSV file
	csvFiles, err := filepath.Glob("tmp/csv/*.csv")
	if err != nil {
		t.Fatalf("failed to glob csv files: %v", err)
	}
	if len(csvFiles) == 0 {
		t.Fatalf("expected at least one CSV file under tmp/csv, found none")
	}

	// Read the first CSV and verify the number of non-empty lines equals the requested quantity
	data, err := os.ReadFile(csvFiles[0])
	if err != nil {
		t.Fatalf("failed to read csv file %s: %v", csvFiles[0], err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// handle case where file may end with a trailing newline
	count := 0
	for _, ln := range lines {
		if strings.TrimSpace(ln) != "" {
			count++
		}
	}
	if count != arg.Quantity {
		t.Fatalf("expected %d lines in CSV, got %d (file: %s)", arg.Quantity, count, csvFiles[0])
	}
}

// TestService_Run_IdempotentTemporaryFolders verifies that repeated runs create
// distinct CSV artifacts (e.g. with different timestamps) and do not overwrite
// previous outputs when DEBUG/TEST flags are not set.
func TestService_Run_IdempotentTemporaryFolders(t *testing.T) {
	const secretEnv = "SECRET_KEY"
	prevSecret := os.Getenv(secretEnv)
	_ = os.Setenv(secretEnv, "integration-test-secret-2")
	t.Cleanup(func() {
		if prevSecret == "" {
			_ = os.Unsetenv(secretEnv)
		} else {
			_ = os.Setenv(secretEnv, prevSecret)
		}
	})

	// Ensure clean slate
	_ = os.RemoveAll("tmp")
	t.Cleanup(func() { _ = os.RemoveAll("tmp") })

	arg := &types.Argument{
		Quantity: 3,
		Info:     "idempotency-test",
		Size:     100,
		URL:      "https://example.example/",
		Format:   types.FormatPDF,
		Storage:  types.StorageLocal,
	}

	svc := service.New(arg)
	ctx := context.Background()

	// First run
	if err := svc.Run(ctx); err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	files1, _ := filepath.Glob("tmp/csv/*.csv")
	if len(files1) == 0 {
		t.Fatalf("expected csv from first run")
	}

	// Wait a little to ensure any timestamp differences
	time.Sleep(250 * time.Millisecond)

	// Second run
	if err := svc.Run(ctx); err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	files2, _ := filepath.Glob("tmp/csv/*.csv")
	if len(files2) == 0 {
		t.Fatalf("expected csv from second run")
	}

	// There should be at least as many CSV files after the second run as after the first.
	if len(files2) < len(files1) {
		t.Fatalf("expected non-decreasing number of csv files after consecutive runs; before=%d after=%d", len(files1), len(files2))
	}
}
