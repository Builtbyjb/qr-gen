package codegen

import (
	"crypto/sha256"
	"os"
	"strings"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const base = int64(62)

// GenerateBase62 encodes the provided cursor as a fixed-length base62 string.
// If cursor is 0 the result will be the padding character repeated `length` times.
func GenerateBase62(cursor int64, length int) string {
	if length <= 0 {
		return ""
	}

	buf := make([]byte, length)
	last := length - 1
	i := cursor
	if i < 0 {
		i = 0
	}

	for i > 0 && last >= 0 {
		idx := int(i % base)
		buf[last] = base62Chars[idx]
		i = i / base
		last--
	}

	// pad remaining with '0' (first char)
	for last >= 0 {
		buf[last] = base62Chars[0]
		last--
	}

	return string(buf)
}

// GenerateHash computes a short 3-character string derived from the
// SECRET_KEY environment variable. It uses SHA-256 and maps bytes to
// base62 character set, returning an uppercase 3-character string.
//
// This variant panics if SECRET_KEY is not set to keep the API simple
// for callers that expect a deterministic string (matching original behavior).
func GenerateHash() string {
	salt := os.Getenv("SECRET_KEY")
	if strings.TrimSpace(salt) == "" {
		// Fail fast - the rest of the pipeline depends on this secret.
		panic("environment variable SECRET_KEY not found")
	}

	sum := sha256.Sum256([]byte(salt))
	bytes := sum[:] // slice of bytes

	var b strings.Builder
	for i := 0; i < len(bytes) && b.Len() < 3; i++ {
		// map byte value to base62 by modulus
		idx := int(bytes[i]) % len(base62Chars)
		b.WriteByte(base62Chars[idx])
	}

	// Pad to length 3 if necessary
	for b.Len() < 3 {
		b.WriteByte(base62Chars[0])
	}

	return strings.ToUpper(b.String())
}

// RandomInsertHash inserts the `hash` into the `qr` string at a position
// determined by cursor. The strategy mirrors the original implementation:
// - position 0: take qr[2:7] + hash + qr[0:2]
// - position 1: qr[0:3] + hash + qr[3:7]
// - position 2: qr[3:7] + hash + qr[0:3]
// - position 4: qr + hash
// - default: hash + qr
//
// If the qr string is shorter than expected, the function falls back
// to a safe concatenation.
func RandomInsertHash(qr, hash string, cursor int64) string {
	if cursor < 0 {
		cursor = -cursor
	}
	pos := int(cursor % 5)

	// Ensure we don't panic on short strings; use safe slicing
	switch pos {
	case 0:
		if len(qr) >= 7 {
			return qr[2:7] + hash + qr[0:2]
		}
	case 1:
		if len(qr) >= 7 {
			return qr[0:3] + hash + qr[3:7]
		}
	case 2:
		if len(qr) >= 7 {
			return qr[3:7] + hash + qr[0:3]
		}
	case 4:
		return qr + hash
	// case 5 in original was unreachable when using %5; treat others as default
	default:
		return hash + qr
	}

	// Fallback safe concatenation
	return hash + qr
}

// GenerateQRCode composes a QR-like code string for the given cursor.
// It internally generates a base62 string of length 7, derives a short hash
// from SECRET_KEY and inserts it based on the cursor. The returned code
// is prefixed with "QR-".
//
// This function returns the code directly and will panic if SECRET_KEY is
// missing (via GenerateHash), matching a simpler, caller-friendly signature.
func GenerateQRCode(cursor int64) string {
	base := GenerateBase62(cursor, 7)
	h := GenerateHash()
	qrWithHash := RandomInsertHash(base, h, cursor)
	return "QR-" + qrWithHash
}
