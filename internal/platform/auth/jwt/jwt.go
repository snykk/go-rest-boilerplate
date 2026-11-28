// Package jwt issues and verifies signed access tokens: JSON Web Tokens signed with HS256. An Issuer signs a
// token for a subject and its roles, and a Verifier, an auth.Authenticator, turns a token back into the
// auth.Principal it names.
package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
)

// MinSecretLength is the fewest bytes a Config's secret may have.
const MinSecretLength = 32

// signingMethod is the one algorithm tokens are signed and verified with.
const signingMethod = "HS256"

// Config holds what an Issuer and a Verifier share. The Verifier accepts only tokens an Issuer of the same
// Config signs. NewIssuer and NewVerifier validate it.
type Config struct {
	// Secret is the key tokens are signed with: at least MinSecretLength random bytes, kept secret. It stays a
	// config.Secret, so printing a Config or a component built from it never shows the key.
	Secret config.Secret
	// Issuer is the "iss" claim of a token, such as the name of the service that signs in. It must not be empty.
	Issuer string
	// Audience is the "aud" claim of a token, such as the name of the API that accepts it. It must not be empty.
	Audience string
	// TTL is the lifetime of a token the Issuer signs, and must be positive for an Issuer. The Verifier does not
	// read it.
	TTL time.Duration
	// Leeway is how far the Verifier tolerates clock drift between the signer and itself when it checks the
	// expiry of a token. Zero allows none; it must not be negative for a Verifier.
	Leeway time.Duration
}

type claims struct {
	jwtlib.RegisteredClaims
	Roles []string `json:"roles,omitempty"`
}

func (c Config) validate() error {
	if len(c.Secret.Reveal()) < MinSecretLength {
		return fmt.Errorf("jwt: the secret must be at least %d bytes", MinSecretLength)
	}
	if c.Issuer == "" {
		return errors.New("jwt: the issuer must not be empty")
	}
	if c.Audience == "" {
		return errors.New("jwt: the audience must not be empty")
	}

	return nil
}
