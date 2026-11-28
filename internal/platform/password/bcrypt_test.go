package password_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/snykk/go-rest-boilerplate/internal/platform/password"
)

// timingCost is the cost of the timing test: above bcrypt's default of 10, so a dummy hash made at a fixed
// default cost instead of the configured one would take a quarter of a real compare or less.
const timingCost = 12

// NewBcrypt accepts the costs bcrypt supports and refuses the others.
func TestNewBcrypt(t *testing.T) {
	tCases := []struct {
		name    string
		cost    int
		wantErr bool
	}{
		{name: "the lowest cost bcrypt supports is accepted", cost: bcrypt.MinCost},
		{name: "a cost below bcrypt's lowest is refused", cost: bcrypt.MinCost - 1, wantErr: true},
		{name: "a cost above bcrypt's highest is refused", cost: bcrypt.MaxCost + 1, wantErr: true},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := password.NewBcrypt(tc.cost)

			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("NewBcrypt(%d) = %v, want error: %t", tc.cost, err, tc.wantErr)
			}
			if tc.wantErr && got != nil {
				t.Errorf("NewBcrypt(%d) = %v, want nil with the error", tc.cost, got)
			}
		})
	}
}

// Hash returns a salted bcrypt hash of the password at the configured cost, never the password itself, and
// refuses a password longer than bcrypt reads without repeating it.
func TestBcryptHash(t *testing.T) {
	hasher := newBcrypt(t, bcrypt.MinCost)
	tCases := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "a password gets a salted hash at the cost", password: "correct horse"},
		{name: "a password of 73 bytes is refused", password: strings.Repeat("p", 73), wantErr: true},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			first, err := hasher.Hash(tc.password)

			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("Hash = %v, want error: %t", err, tc.wantErr)
			}
			if tc.wantErr {
				if strings.Contains(err.Error(), tc.password) {
					t.Errorf("Hash error = %q, want it without the password", err)
				}
				return
			}
			if cost, err := bcrypt.Cost([]byte(first)); err != nil || cost != bcrypt.MinCost {
				t.Errorf("bcrypt.Cost(Hash) = %d, %v; want %d", cost, err, bcrypt.MinCost)
			}
			if err := bcrypt.CompareHashAndPassword([]byte(first), []byte(tc.password)); err != nil {
				t.Errorf("CompareHashAndPassword(Hash) = %v, want nil", err)
			}
			if second, err := hasher.Hash(tc.password); err != nil || second == first {
				t.Errorf("Hash twice = %q and %q (%v), want two salts", first, second, err)
			}
		})
	}
}

// Compare matches a hash with its own password only, and a hash it cannot read with no password.
func TestBcryptCompare(t *testing.T) {
	hasher := newBcrypt(t, bcrypt.MinCost)
	hash, err := hasher.Hash("correct horse")
	if err != nil {
		t.Fatalf("Hash = %v", err)
	}
	tCases := []struct {
		name     string
		hash     string
		password string
		want     bool
	}{
		{
			name:     "the right password matches",
			hash:     hash,
			password: "correct horse",
			want:     true,
		},
		{name: "a wrong password does not match", hash: hash, password: "battery staple"},
		{name: "a hash it cannot read matches nothing", hash: "not-a-bcrypt-hash", password: "correct horse"},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasher.Compare(tc.hash, tc.password)

			if got != tc.want {
				t.Errorf("Compare = %t, want %t", got, tc.want)
			}
		})
	}
}

// CompareDummy costs at least half of a real Compare at the configured cost, so a sign-in for an account that
// does not exist takes about as long as one with a wrong password.
func TestBcryptCompareDummy(t *testing.T) {
	hasher := newBcrypt(t, timingCost)
	hash, err := hasher.Hash("correct horse")
	if err != nil {
		t.Fatalf("Hash = %v", err)
	}

	compare := fastest(func() { hasher.Compare(hash, "battery staple") })
	dummy := fastest(func() { hasher.CompareDummy("battery staple") })

	if dummy < compare/2 {
		t.Errorf("CompareDummy took %v, want at least half of Compare's %v", dummy, compare)
	}
}

// newBcrypt is a Bcrypt at cost, failing the test when there is none.
func newBcrypt(t *testing.T, cost int) *password.Bcrypt {
	t.Helper()
	hasher, err := password.NewBcrypt(cost)
	if err != nil {
		t.Fatalf("NewBcrypt(%d) = %v", cost, err)
	}
	return hasher
}

// fastest is the shortest of three runs of call: the run least slowed by the rest of the machine.
func fastest(call func()) time.Duration {
	best := time.Duration(math.MaxInt64)
	for range 3 {
		start := time.Now()
		call()
		best = min(best, time.Since(start))
	}
	return best
}
