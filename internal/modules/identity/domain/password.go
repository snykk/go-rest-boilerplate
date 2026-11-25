package domain

import (
	"fmt"
	"io"
)

const (
	minPasswordBytes = 8
	maxPasswordBytes = 72
)

const redacted = "[redacted]"

// Password is a plain-text password that passed NewPassword. Formatted with fmt, under any verb, alone, through
// a pointer or as an exported field of a struct, it writes "[redacted]" and never its value. A Password held in
// an unexported field of a struct is printed by reflection, which no method can intercept, so keep it out of
// such fields.
type Password struct {
	value string
}

// NewPassword returns text as a Password. It returns ErrWeakPassword unless text is 8 to 72 bytes: bytes, not
// characters, because that is the limit of what bcrypt reads.
func NewPassword(text string) (Password, error) {
	if len(text) < minPasswordBytes || len(text) > maxPasswordBytes {
		return Password{}, ErrWeakPassword
	}
	return Password{value: text}, nil
}

// Reveal returns the password's text, which no other method prints. Treat the result as a secret.
func (p Password) Reveal() string { return p.value }

// String returns "[redacted]".
func (p Password) String() string { return redacted }

// GoString returns "[redacted]".
func (p Password) GoString() string { return redacted }

// Format implements fmt.Formatter: it writes "[redacted]" whatever the verb and flags.
func (p Password) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, redacted)
}
