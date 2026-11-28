package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// Issuer signs access tokens. It is safe for concurrent use.
type Issuer struct {
	config Config
	now    func() time.Time
}

// NewIssuer returns an Issuer of config that reads the current time from now. It returns an error, which
// never contains the secret, when the secret is shorter than MinSecretLength bytes, the issuer or the audience
// is empty, the TTL is not positive, or now is nil.
func NewIssuer(config Config, now func() time.Time) (*Issuer, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	if config.TTL <= 0 {
		return nil, errors.New("jwt: the token lifetime must be positive")
	}
	if now == nil {
		return nil, errors.New("jwt: the clock must not be nil")
	}

	return &Issuer{config: config, now: now}, nil
}

// Issue returns an HS256 token for subject, such as a user ID, with roles, and the time it expires, which is
// the Config's TTL after now, in whole seconds: the value of the exp claim. The token carries the claims sub
// (subject), roles, iss, aud, iat and exp, the times in whole seconds. It returns an error when subject is
// empty or signing fails.
func (i *Issuer) Issue(subject string, roles []string) (token string, expiresAt time.Time, err error) {
	if subject == "" {
		return "", time.Time{}, errors.New("jwt: the subject must not be empty")
	}

	now := i.now()
	expiresAt = now.Add(i.config.TTL).Truncate(time.Second)
	tokenClaims := claims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   subject,
			Issuer:    i.config.Issuer,
			Audience:  jwtlib.ClaimStrings{i.config.Audience},
			IssuedAt:  jwtlib.NewNumericDate(now.Truncate(time.Second)),
			ExpiresAt: jwtlib.NewNumericDate(expiresAt),
		},
		Roles: roles,
	}

	unsigned := jwtlib.NewWithClaims(jwtlib.GetSigningMethod(signingMethod), tokenClaims)
	token, err = unsigned.SignedString([]byte(i.config.Secret.Reveal()))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("jwt: sign the token: %w", err)
	}

	return token, expiresAt, nil
}
