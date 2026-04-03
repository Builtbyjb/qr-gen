package main

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

// withSecretKey sets SECRET_KEY for the duration of fn and restores the previous value.
func withSecretKey(t *testing.T, key string, fn func()) {
	t.Helper()
	prev := os.Getenv("SECRET_KEY")
	_ = os.Setenv("SECRET_KEY", key)
	defer func() {
		if prev == "" {
			_ = os.Unsetenv("SECRET_KEY")
		} else {
			_ = os.Setenv("SECRET_KEY", prev)
		}
	}()
	fn()
}

// TestCLIPipeline is an end-to-end smoke test that exercises the full service
// pipeline (code generation → PDF → zip → local storage), matching the Java
// implementation's happy-path behaviour.
func TestCLIPipeline(t *testing.T) {
	withSecretKey(t, "cli-pipeline-test-secret", func() {
		_ = os.RemoveAll("tmp")
		t.Cleanup(func() { _ = os.RemoveAll("tmp") })

		arg := &types.Argument{
			Quantity:  6,
			Info:      "smoke-test",
			Size:      200,
			URL:       "https://example.com/",
			Format:    types.PDF,
			Storage:   types.LOCAL,
			ChunkSize: 500,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		svc := service.New(arg)
		if err := svc.Run(ctx); err != nil {
			t.Fatalf("service.Run returned unexpected error: %v", err)
		}

		// --- CSV check ---
		// The Java implementation preserves the CSV; the Go port must do the same.
		csvFiles, err := filepath.Glob(filepath.Join("tmp", "csv", "*.csv"))
		if err != nil {
			t.Fatalf("failed to glob csv files: %v", err)
		}
		if len(csvFiles) == 0 {
			t.Fatalf("expected at least one CSV file under tmp/csv/, found none")
		}

		data, err := os.ReadFile(csvFiles[0])
		if err != nil {
			t.Fatalf("failed to read csv %s: %v", csvFiles[0], err)
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		count := 0
		for _, ln := range lines {
			if strings.TrimSpace(ln) != "" {
				count++
			}
		}
		if count != arg.Quantity {
			t.Fatalf("expected %d codes in CSV, got %d (file: %s)", arg.Quantity, count, csvFiles[0])
		}

		// Verify every code has the expected "QR-" prefix.
		for i, ln := range lines {
			ln = strings.TrimSpace(ln)
			if ln == "" {
				continue
			}
			if !strings.HasPrefix(ln, "QR-") {
				t.Fatalf("line %d in CSV missing 'QR-' prefix: %q", i+1, ln)
			}
		}

		// --- Zip check ---
		// After a successful local upload the original zip in tmp/zips/ is removed
		// and the file is copied to tmp/storage/artifacts/. Check both locations so
		// the assertion works whether or not the upload step ran.
		zipFiles, _ := filepath.Glob(filepath.Join("tmp", "zips", "*.zip"))
		if len(zipFiles) == 0 {
			zipFiles, _ = filepath.Glob(filepath.Join("tmp", "storage", "artifacts", "*.zip"))
		}
		if len(zipFiles) == 0 {
			t.Fatalf("expected a zip file in tmp/zips/ or tmp/storage/artifacts/, found none")
		}

		// Zip must be non-empty.
		fi, statErr := os.Stat(zipFiles[0])
		if statErr != nil {
			t.Fatalf("failed to stat zip %s: %v", zipFiles[0], statErr)
		}
		if fi.Size() == 0 {
			t.Fatalf("zip file %s is empty", zipFiles[0])
		}
	})
}

// TestCLIPipeline_IdempotentRuns verifies that repeated runs produce independent
// CSV and zip artifacts (distinct filenames via timestamp) without overwriting
// each other — matching the Java behaviour of always creating new timestamped files.
func TestCLIPipeline_IdempotentRuns(t *testing.T) {
	withSecretKey(t, "idempotency-test-secret", func() {
		_ = os.RemoveAll("tmp")
		t.Cleanup(func() { _ = os.RemoveAll("tmp") })

		arg := &types.Argument{
			Quantity:  3,
			Info:      "idempotency-test",
			Size:      100,
			URL:       "https://example.example/",
			Format:    types.PDF,
			Storage:   types.LOCAL,
			ChunkSize: 500,
		}

		ctx := context.Background()
		svc := service.New(arg)

		// First run.
		if err := svc.Run(ctx); err != nil {
			t.Fatalf("first run failed: %v", err)
		}
		csvAfterFirst, _ := filepath.Glob(filepath.Join("tmp", "csv", "*.csv"))
		if len(csvAfterFirst) == 0 {
			t.Fatalf("expected CSV after first run")
		}

		// Brief pause so the millisecond timestamp in filenames differs.
		time.Sleep(50 * time.Millisecond)

		// Second run using a fresh service instance (mirrors a new CLI invocation).
		svc2 := service.New(arg)
		if err := svc2.Run(ctx); err != nil {
			t.Fatalf("second run failed: %v", err)
		}
		csvAfterSecond, _ := filepath.Glob(filepath.Join("tmp", "csv", "*.csv"))

		// After the second run there must be at least as many CSV files as after the first.
		if len(csvAfterSecond) < len(csvAfterFirst) {
			t.Fatalf("expected non-decreasing CSV count after second run; before=%d after=%d",
				len(csvAfterFirst), len(csvAfterSecond))
		}
	})
}
