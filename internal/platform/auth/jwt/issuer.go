package jwt

import (
	"errors"
	"fmt"
	"slices"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// Issuer signs access tokens. It is safe for concurrent use.
type Issuer struct {
	config Config
	now    func() time.Time
}

// NewIssuer returns an Issuer of config that reads the current time from now. It keeps its own copy of the
// secret.
func NewIssuer(config Config, now func() time.Time) *Issuer {
	config.Secret = slices.Clone(config.Secret)
	return &Issuer{config: config, now: now}
}

// Issue returns an HS256 token for subject, such as a user ID, with roles, and the time it expires, which is
// the Config's TTL after now. The token carries the claims sub (subject), roles, iss, aud, iat and exp, the
// times in whole seconds. It returns an error when the TTL is not positive and when the Config lacks a secret,
// an issuer or an audience.
func (i *Issuer) Issue(subject string, roles []string) (token string, expiresAt time.Time, err error) {
	if i.config.TTL <= 0 {
		return "", time.Time{}, errors.New("jwt: the token lifetime must be positive")
	}
	if !i.config.complete() {
		return "", time.Time{}, errors.New("jwt: the configuration needs a secret, an issuer and an audience")
	}

	now := i.now()
	expiresAt = now.Add(i.config.TTL)
	tokenClaims := claims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   subject,
			Issuer:    i.config.Issuer,
			Audience:  jwtlib.ClaimStrings{i.config.Audience},
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(expiresAt),
		},
		Roles: roles,
	}

	unsigned := jwtlib.NewWithClaims(jwtlib.GetSigningMethod(signingMethod), tokenClaims)
	token, err = unsigned.SignedString(i.config.Secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("jwt: sign the token: %w", err)
	}

	return token, expiresAt, nil
}
