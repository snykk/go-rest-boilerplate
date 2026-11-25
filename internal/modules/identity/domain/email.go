package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxEmailLen = 254

// Email is a normalized e-mail address: trimmed, lower-cased and free of the characters that could break out
// of an SMTP envelope. Its zero value is the empty address, which no constructor returns.
type Email struct {
	value string
}

// NewEmail returns address trimmed and lower-cased as an Email. It returns ErrInvalidEmail unless the result is
// at most 254 bytes of valid UTF-8 with one @ between a non-empty local part and a non-empty domain, and with
// no whitespace, double quote, angle bracket or control character anywhere.
func NewEmail(address string) (Email, error) {
	address = strings.TrimSpace(address)
	if !utf8.ValidString(address) {
		return Email{}, ErrInvalidEmail
	}
	address = strings.ToLower(address)
	if len(address) > maxEmailLen || strings.ContainsFunc(address, forbiddenInAddress) {
		return Email{}, ErrInvalidEmail
	}
	local, host, found := strings.Cut(address, "@")
	if !found || local == "" || host == "" || strings.Contains(host, "@") {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: address}, nil
}

// String returns the normalized address.
func (e Email) String() string { return e.value }

func forbiddenInAddress(character rune) bool {
	return unicode.IsSpace(character) || unicode.IsControl(character) ||
		character == '"' || character == '<' || character == '>'
}
