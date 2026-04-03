package context

import (
	"os"
	"strconv"
	"sync"
)

// ContextVar is a small helper for reading integer environment variables
// with a default value. It mimics the idea of a process-level context flag
// (e.g. DEBUG, TEST) that can be created once and referenced throughout the app.
type ContextVar struct {
	key   string
	value int
}

var (
	// cache keeps track of created ContextVars to avoid accidental recreation.
	cache   = make(map[string]*ContextVar)
	cacheMu sync.Mutex
)

// NewContextVar creates a ContextVar for the given key and default value.
// If the same key was previously created, this function will panic to avoid
// accidental redefinition (matching the stricter behavior found in some ports).
// The returned ContextVar reads the environment variable named `key` (if present)
// and parses it as an integer; if parsing fails or the env var is empty the
// provided defaultValue is used instead.
func NewContextVar(key string, defaultValue int) *ContextVar {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if _, exists := cache[key]; exists {
		panic("attempt to recreate ContextVar " + key)
	}

	cv := &ContextVar{
		key:   key,
		value: defaultValue,
	}

	if v := os.Getenv(key); v != "" {
		if iv, err := strconv.Atoi(v); err == nil {
			cv.value = iv
		}
	}

	cache[key] = cv
	return cv
}

// GetKey returns the environment variable key associated with this ContextVar.
func (c *ContextVar) GetKey() string {
	return c.key
}

// GetValue returns the integer value of this ContextVar.
func (c *ContextVar) GetValue() int {
	return c.value
}

// GetAsBool interprets the integer value as a boolean (0 => false, non-zero => true).
func (c *ContextVar) GetAsBool() bool {
	return c.value != 0
}

// GreaterThan returns true if this ContextVar's value is > x.
func (c *ContextVar) GreaterThan(x int) bool {
	return c.value > x
}

// GreaterThanOrEqual returns true if this ContextVar's value is >= x.
func (c *ContextVar) GreaterThanOrEqual(x int) bool {
	return c.value >= x
}

// LessThan returns true if this ContextVar's value is < x.
func (c *ContextVar) LessThan(x int) bool {
	return c.value < x
}

// LessThanOrEqual returns true if this ContextVar's value is <= x.
func (c *ContextVar) LessThanOrEqual(x int) bool {
	return c.value <= x
}

// EqualsValue returns true if this ContextVar's value equals x.
func (c *ContextVar) EqualsValue(x int) bool {
	return c.value == x
}

// String returns a compact debugging representation.
func (c *ContextVar) String() string {
	return "ContextVar(key='" + c.key + "', value=" + strconv.Itoa(c.value) + ")"
}
