// Package jwt issues and verifies signed access tokens: JSON Web Tokens signed with HS256. An Issuer signs a
// token for a subject and its roles, and a Verifier, an auth.Authenticator, turns a token back into the
// auth.Principal it names.
package jwt

import (
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// signingMethod is the one algorithm tokens are signed and verified with.
const signingMethod = "HS256"

// Config holds what an Issuer and a Verifier share. The Verifier accepts only tokens an Issuer of the same
// Config signs.
type Config struct {
	// Secret is the key tokens are signed with: at least 32 random bytes, kept secret. The Issuer and the
	// Verifier refuse to work without one.
	Secret []byte
	// Issuer is the "iss" claim of a token, such as the name of the service that signs in.
	Issuer string
	// Audience is the "aud" claim of a token, such as the name of the API that accepts it.
	Audience string
	// TTL is the lifetime of a token the Issuer signs. The Issuer refuses to sign when it is not positive; the
	// Verifier does not read it.
	TTL time.Duration
	// Leeway is how far the Verifier tolerates clock drift between the signer and itself when it checks the
	// expiry of a token. Zero allows none.
	Leeway time.Duration
}

func (c Config) complete() bool {
	return len(c.Secret) > 0 && c.Issuer != "" && c.Audience != ""
}

type claims struct {
	jwtlib.RegisteredClaims
	Roles []string `json:"roles,omitempty"`
}
