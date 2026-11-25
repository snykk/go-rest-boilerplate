package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// NewEmail trims and lower-cases an address, and refuses one that is not a single local part and domain
// joined by one @, is longer than 254 bytes, or holds a character that could break an SMTP envelope.
func TestNewEmail(t *testing.T) {
	longest := strings.Repeat("a", 249) + "@x.co"
	tCases := []struct {
		name    string
		in      string
		want    string
		wantErr error
	}{
		{name: "a plain address is kept", in: "ada@example.com", want: "ada@example.com"},
		{
			name: "mixed case and surrounding spaces are normalized",
			in:   "  Ada.Lovelace@Example.COM \t",
			want: "ada.lovelace@example.com",
		},
		{name: "an address of 254 bytes is accepted", in: longest, want: longest},
		{
			name:    "an address of 255 bytes is refused",
			in:      "a" + longest,
			wantErr: domain.ErrInvalidEmail,
		},
		{name: "an empty address is refused", in: "", wantErr: domain.ErrInvalidEmail},
		{name: "a blank address is refused", in: "   ", wantErr: domain.ErrInvalidEmail},
		{name: "an address with no @ is refused", in: "ada.example.com", wantErr: domain.ErrInvalidEmail},
		{name: "an address with two @ is refused", in: "ada@@example.com", wantErr: domain.ErrInvalidEmail},
		{name: "an address with two @ apart is refused", in: "ada@x@example.com", wantErr: domain.ErrInvalidEmail},
		{name: "an empty local part is refused", in: "@example.com", wantErr: domain.ErrInvalidEmail},
		{name: "an empty domain is refused", in: "ada@", wantErr: domain.ErrInvalidEmail},
		{name: "a space in the local part is refused", in: "ada lovelace@example.com", wantErr: domain.ErrInvalidEmail},
		{name: "a tab in the local part is refused", in: "ada\t@example.com", wantErr: domain.ErrInvalidEmail},
		{name: "a no-break space is refused", in: "ada @example.com", wantErr: domain.ErrInvalidEmail},
		{name: "a space in the domain is refused", in: "ada@example .com", wantErr: domain.ErrInvalidEmail},
		{name: "a quoted local part is refused", in: `"ada"@example.com`, wantErr: domain.ErrInvalidEmail},
		{
			name:    "a quoted local part holding a space is refused",
			in:      `"ada lovelace"@example.com`,
			wantErr: domain.ErrInvalidEmail,
		},
		{name: "a double quote in the domain is refused", in: `ada@exam"ple.com`, wantErr: domain.ErrInvalidEmail},
		{name: "an opening angle bracket is refused", in: "<ada@example.com", wantErr: domain.ErrInvalidEmail},
		{name: "a closing angle bracket is refused", in: "ada@example.com>", wantErr: domain.ErrInvalidEmail},
		{
			name:    "a line break followed by a command is refused",
			in:      "ada@example.com\r\nRCPT TO:x",
			wantErr: domain.ErrInvalidEmail,
		},
		{name: "a NUL character is refused", in: "ada\x00@example.com", wantErr: domain.ErrInvalidEmail},
		{name: "a DEL character is refused", in: "ada\x7f@example.com", wantErr: domain.ErrInvalidEmail},
		{name: "bytes that are not UTF-8 are refused", in: "ada\xff@example.com", wantErr: domain.ErrInvalidEmail},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.NewEmail(tc.in)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("NewEmail(%q) = %v, want %v", tc.in, err, tc.wantErr)
			}
			if got.String() != tc.want {
				t.Errorf("NewEmail(%q) = %q, want %q", tc.in, got.String(), tc.want)
			}
		})
	}
}
