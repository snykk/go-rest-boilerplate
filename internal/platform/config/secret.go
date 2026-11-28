package config

import (
	"fmt"
	"log/slog"
)

const redacted = "[REDACTED]"

// Secret is a setting that must never be printed, such as a password, a token, a key or a connection string
// with credentials. Formatting it with fmt, logging it with log/slog and encoding it as JSON or text never
// show the value: the secret itself, or an exported field that holds it, gives "[REDACTED]", or the empty
// string for an empty secret; through an unexported field they reach only a pointer, which fmt and slog's text
// handler print and JSON leaves out. Only Reveal returns the value. The zero Secret is empty. A Secret is
// immutable and safe for concurrent use; compare secrets with a constant-time function on their revealed
// values, never with ==.
type Secret struct{ value *string }

// NewSecret returns a Secret holding value; an empty value gives the zero Secret.
func NewSecret(value string) Secret {
	if value == "" {
		return Secret{}
	}
	return Secret{value: &value}
}

// Reveal returns the value, or the empty string for the zero Secret. Call it only where the value is handed to
// the library that uses it, or where a rule on the value, such as a minimum length, is checked.
func (s Secret) Reveal() string {
	if s.value == nil {
		return ""
	}
	return *s.value
}

// String returns "[REDACTED]", or the empty string for an empty secret.
func (s Secret) String() string {
	if s.value == nil {
		return ""
	}
	return redacted
}

// Format writes String for every verb, so no verb reaches the value.
func (s Secret) Format(f fmt.State, _ rune) {
	_, _ = f.Write([]byte(s.String()))
}

// LogValue returns String as the logged value.
func (s Secret) LogValue() slog.Value { return slog.StringValue(s.String()) }

// MarshalText returns String, which json and other text encoders write in place of the value.
func (s Secret) MarshalText() ([]byte, error) { return []byte(s.String()), nil }
