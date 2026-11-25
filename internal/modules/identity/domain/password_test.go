package domain_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// NewPassword accepts a password of 8 to 72 bytes, counted in bytes because that is what bcrypt reads, and
// refuses any other length.
func TestNewPassword(t *testing.T) {
	tCases := []struct {
		name    string
		in      string
		wantErr error
	}{
		{name: "eight bytes is enough", in: strings.Repeat("a", 8)},
		{name: "seven bytes is refused", in: strings.Repeat("a", 7), wantErr: domain.ErrWeakPassword},
		{name: "72 bytes is the most", in: strings.Repeat("a", 72)},
		{name: "73 bytes is refused", in: strings.Repeat("a", 73), wantErr: domain.ErrWeakPassword},
		{name: "an empty password is refused", in: "", wantErr: domain.ErrWeakPassword},
		{name: "spaces count like any other byte", in: strings.Repeat(" ", 8)},
		{name: "a password of 36 two-byte characters is 72 bytes and accepted", in: strings.Repeat("é", 36)},
		{
			name:    "a password of 40 characters but 80 bytes is refused",
			in:      strings.Repeat("é", 40),
			wantErr: domain.ErrWeakPassword,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.NewPassword(tc.in)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("NewPassword(%d bytes) = %v, want %v", len(tc.in), err, tc.wantErr)
			}
			if tc.wantErr == nil && got.Reveal() != tc.in {
				t.Errorf("NewPassword(%d bytes).Reveal() differs from the input, want the same text", len(tc.in))
			}
		})
	}
}

// Password writes "[redacted]" under every fmt verb, alone, through a pointer and as an exported field of a
// struct, and never its value.
func TestPasswordFormat(t *testing.T) {
	const secret = "correct horse battery"
	password, err := domain.NewPassword(secret)
	if err != nil {
		t.Fatalf("NewPassword = %v", err)
	}
	holder := struct{ P domain.Password }{password}
	tCases := []struct {
		name   string
		format string
		arg    any
		want   string
	}{
		{name: "verb v", format: "%v", arg: password, want: "[redacted]"},
		{name: "verb plus v", format: "%+v", arg: password, want: "[redacted]"},
		{name: "verb sharp v", format: "%#v", arg: password, want: "[redacted]"},
		{name: "verb s", format: "%s", arg: password, want: "[redacted]"},
		{name: "verb q", format: "%q", arg: password, want: "[redacted]"},
		{name: "verb d", format: "%d", arg: password, want: "[redacted]"},
		{name: "verb x", format: "%x", arg: password, want: "[redacted]"},
		{name: "verb upper x", format: "%X", arg: password, want: "[redacted]"},
		{name: "verb t", format: "%t", arg: password, want: "[redacted]"},
		{name: "verb f", format: "%f", arg: password, want: "[redacted]"},
		{name: "verb e", format: "%e", arg: password, want: "[redacted]"},
		{name: "a verb with width and flags", format: "%-30.5s", arg: password, want: "[redacted]"},
		{name: "a pointer with verb v", format: "%v", arg: &password, want: "[redacted]"},
		{name: "a pointer with verb d", format: "%d", arg: &password, want: "[redacted]"},
		{name: "a struct field with verb v", format: "%v", arg: holder, want: "{[redacted]}"},
		{name: "a struct field with verb plus v", format: "%+v", arg: holder, want: "{P:[redacted]}"},
		{name: "a struct field with verb d", format: "%d", arg: holder, want: "{[redacted]}"},
		{name: "a struct field with verb x", format: "%x", arg: holder, want: "{[redacted]}"},
		{name: "a struct field with verb q", format: "%q", arg: holder, want: "{[redacted]}"},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got := fmt.Sprintf(tc.format, tc.arg)

			if got != tc.want {
				t.Errorf("Sprintf(%q, password) = %q, want %q", tc.format, got, tc.want)
			}
		})
	}

	if got := fmt.Sprint(password); got != "[redacted]" {
		t.Errorf("Sprint(password) = %q, want %q", got, "[redacted]")
	}
	if got := password.String(); got != "[redacted]" {
		t.Errorf("String() = %q, want %q", got, "[redacted]")
	}
	if got := password.GoString(); got != "[redacted]" {
		t.Errorf("GoString() = %q, want %q", got, "[redacted]")
	}
}
