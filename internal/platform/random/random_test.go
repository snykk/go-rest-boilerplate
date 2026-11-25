package random_test

import (
	"encoding/base64"
	"regexp"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/random"
)

// draws is how many values each test draws.
const draws = 10_000

// Code returns six decimal digits, leading zeros kept, spread over the million codes: every first digit,
// zero included, comes up about a tenth of the time, and few codes come up twice.
func TestGeneratorCode(t *testing.T) {
	sixDigits := regexp.MustCompile(`^[0-9]{6}$`)
	seen := map[string]bool{}
	var firstDigits [10]int

	for range draws {
		code := random.Generator{}.Code()

		if !sixDigits.MatchString(code) {
			t.Fatalf("Code = %q, want six digits", code)
		}
		seen[code] = true
		firstDigits[code[0]-'0']++
	}

	for digit, count := range firstDigits {
		if count < draws/10*7/10 || count > draws/10*13/10 {
			t.Errorf("Code started with %d in %d of %d draws, want about a tenth", digit, count, draws)
		}
	}
	if len(seen) < draws-200 {
		t.Errorf("Code gave %d different codes in %d draws, want nearly all different", len(seen), draws)
	}
}

// Token returns 32 random bytes in unpadded base64url, 43 characters, and never the same token twice.
func TestGeneratorToken(t *testing.T) {
	seen := map[string]bool{}

	for range draws {
		token := random.Generator{}.Token()

		raw, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil || len(raw) != 32 || len(token) != 43 {
			t.Fatalf("Token = %q (%d bytes, %v), want 32 bytes in 43 characters of unpadded base64url",
				token, len(raw), err)
		}
		if seen[token] {
			t.Fatalf("Token = %q twice", token)
		}
		seen[token] = true
	}
}
