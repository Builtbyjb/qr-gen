package parser

import (
	"strings"
	"testing"

	"github.com/yourusername/qrgen/pkg/types"
)

func TestParseTime(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0.5, "500.00 μs"},
		{500, "500.00 ms"},
		{1000, "1.00 s"},
		{60000, "1 min 0 s"},
		{3600000, "1 h 0 min"},
		{86400000, "1.00 days"},
	}

	for _, tt := range tests {
		got := ParseTime(tt.in)
		if got != tt.want {
			t.Fatalf("ParseTime(%v) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseArgs_Valid(t *testing.T) {
	args := []string{
		"--quantity=10",
		"--info=Test QR Code",
		"--size=500",
		"--url=https://example.com",
		"--format=pdf",
		"--storage=local",
	}

	arg, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error parsing args: %v", err)
	}

	if arg.Quantity != 10 {
		t.Fatalf("expected quantity 10, got %d", arg.Quantity)
	}
	if arg.Info != "Test QR Code" {
		t.Fatalf("expected info %q, got %q", "Test QR Code", arg.Info)
	}
	if arg.Size != 500 {
		t.Fatalf("expected size 500, got %d", arg.Size)
	}
	if arg.URL != "https://example.com" {
		t.Fatalf("expected url %q, got %q", "https://example.com", arg.URL)
	}

	// Compare Format via FormatFromString to avoid depending on constant names
	expectedFormat, ferr := types.FormatFromString("pdf")
	if ferr != nil {
		t.Fatalf("unexpected error creating expected format: %v", ferr)
	}
	if arg.Format != expectedFormat {
		t.Fatalf("expected format %v, got %v", expectedFormat, arg.Format)
	}

	expectedStorage, serr := types.StorageFromString("local")
	if serr != nil {
		t.Fatalf("unexpected error creating expected storage: %v", serr)
	}
	if arg.Storage != expectedStorage {
		t.Fatalf("expected storage %v, got %v", expectedStorage, arg.Storage)
	}
}

func TestParseArgs_SpecialFlags(t *testing.T) {
	// --help should return an error with message "help"
	_, err := ParseArgs([]string{"--help"})
	if err == nil || !strings.Contains(err.Error(), "help") {
		t.Fatalf("expected help error, got %v", err)
	}

	// --version should return an error with message "version"
	_, err = ParseArgs([]string{"--version"})
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("expected version error, got %v", err)
	}
}

func TestParseArgs_Errors(t *testing.T) {
	// Missing quantity
	_, err := ParseArgs([]string{"--url=https://example.com"})
	if err == nil {
		t.Fatalf("expected error for missing quantity")
	}

	// Missing url
	_, err = ParseArgs([]string{"--quantity=5"})
	if err == nil {
		t.Fatalf("expected error for missing url")
	}

	// Unsupported format (only pdf allowed by parser)
	_, err = ParseArgs([]string{"--quantity=1", "--url=https://x", "--format=png"})
	if err == nil {
		t.Fatalf("expected error for unsupported format")
	}
}
