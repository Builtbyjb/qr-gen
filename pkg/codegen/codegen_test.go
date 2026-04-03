package codegen

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func withSecretKey(t *testing.T, key string, fn func()) {
	t.Helper()
	prev := os.Getenv("SECRET_KEY")
	_ = os.Setenv("SECRET_KEY", key)
	defer func() {
		_ = os.Setenv("SECRET_KEY", prev)
	}()
	fn()
}

func TestGenerateBase62_LengthAndZeroCursor(t *testing.T) {
	tests := []struct {
		cursor int64
		length int
	}{
		{cursor: 0, length: 5},
		{cursor: 1, length: 3},
		{cursor: 12345, length: 7},
	}

	for _, tt := range tests {
		out := GenerateBase62(tt.cursor, tt.length)
		if len(out) != tt.length {
			t.Fatalf("expected length %d, got %d (out=%q) for cursor %d", tt.length, len(out), out, tt.cursor)
		}
	}

	// For cursor 0 we expect padding of the base[0] character (implementation uses '0')
	out0 := GenerateBase62(0, 6)
	if out0 != strings.Repeat("0", 6) {
		t.Fatalf("expected all-zero padding for cursor 0, got %q", out0)
	}
}

func TestGenerateHash_Format(t *testing.T) {
	withSecretKey(t, "unit-test-secret-xyz", func() {
		h := GenerateHash()
		if len(h) != 3 {
			t.Fatalf("expected hash length 3, got %d (%q)", len(h), h)
		}
		// Expect uppercase alphanumeric (based on base62 mapping + ToUpper in generator)
		ok, _ := regexp.MatchString("^[0-9A-Z]{3}$", h)
		if !ok {
			t.Fatalf("hash has unexpected format: %q", h)
		}
	})
}

func TestRandomInsertHash_DoesNotPanicAndContainsHash(t *testing.T) {
	qr := "ABCDEFG" // 7 chars
	hash := "XYZ"

	// try several cursor values covering switch cases (0..5)
	for c := int64(0); c <= 6; c++ {
		out := RandomInsertHash(qr, hash, c)
		if out == "" {
			t.Fatalf("random insert returned empty for cursor %d", c)
		}
		// should contain the hash somewhere
		if !strings.Contains(out, hash) {
			t.Fatalf("result for cursor %d does not contain hash: %q", c, out)
		}
	}
}

func TestGenerateQRCode_UniqueAndPrefix(t *testing.T) {
	withSecretKey(t, "another-secret-for-test", func() {
		set := make(map[string]struct{})
		for i := int64(0); i < 10; i++ {
			code := GenerateQRCode(i)
			if !strings.HasPrefix(code, "QR-") {
				t.Fatalf("expected prefix QR- for code %q", code)
			}
			if _, exists := set[code]; exists {
				t.Fatalf("duplicate code generated for cursor %d: %q", i, code)
			}
			set[code] = struct{}{}
		}
	})
}
