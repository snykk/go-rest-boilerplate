// Package password hashes and compares passwords with bcrypt, and compares against a dummy hash so a sign-in
// for an account that does not exist takes as long as one for an account that does.
package password

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// dummyPassword is what the dummy hash is a hash of. It is no secret: no account has the dummy hash.
const dummyPassword = "no account has this password"

// Bcrypt hashes passwords with bcrypt at one cost. It is safe for concurrent use.
type Bcrypt struct {
	cost  int
	dummy []byte
}

// NewBcrypt returns a Bcrypt that hashes at cost, with the dummy hash CompareDummy compares against made at
// the same cost, once, now. Each step up in cost doubles the time of a hash and of a compare. It returns an
// error when cost is outside bcrypt.MinCost to bcrypt.MaxCost.
func NewBcrypt(cost int) (*Bcrypt, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, fmt.Errorf("password: bcrypt cost %d is outside %d to %d", cost, bcrypt.MinCost, bcrypt.MaxCost)
	}

	dummy, err := bcrypt.GenerateFromPassword([]byte(dummyPassword), cost)
	if err != nil {
		return nil, fmt.Errorf("password: make the dummy hash: %w", err)
	}
	return &Bcrypt{cost: cost, dummy: dummy}, nil
}

// Hash returns the bcrypt hash of password at the Bcrypt's cost, with a random salt. It returns an error when
// bcrypt cannot make one, such as for a password longer than 72 bytes. The error never repeats the password.
func (b *Bcrypt) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", fmt.Errorf("password: hash: %w", err)
	}
	return string(hash), nil
}

// Compare reports whether hash is a bcrypt hash of password, at whatever cost hash was made. A hash it cannot
// read does not match.
func (b *Bcrypt) Compare(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// CompareDummy compares password against the dummy hash and discards the result. It costs what Compare costs
// for a hash made at the Bcrypt's cost.
func (b *Bcrypt) CompareDummy(password string) {
	_ = bcrypt.CompareHashAndPassword(b.dummy, []byte(password))
}
