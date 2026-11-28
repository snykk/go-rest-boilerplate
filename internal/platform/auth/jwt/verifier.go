package jwt

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
)

// Verifier identifies the caller of a request from the access token an Issuer signed. It is an
// auth.Authenticator and is safe for concurrent use.
type Verifier struct {
	secret config.Secret
	parser *jwtlib.Parser
}

// NewVerifier returns a Verifier of config. It returns an error, which never contains the secret, when the
// secret is shorter than MinSecretLength bytes, the issuer or the audience is empty, or the leeway is negative.
func NewVerifier(config Config) (*Verifier, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	if config.Leeway < 0 {
		return nil, errors.New("jwt: the leeway must not be negative")
	}

	return &Verifier{
		secret: config.Secret,
		parser: jwtlib.NewParser(
			jwtlib.WithValidMethods([]string{signingMethod}),
			jwtlib.WithIssuer(config.Issuer),
			jwtlib.WithAudience(config.Audience),
			jwtlib.WithExpirationRequired(),
			jwtlib.WithLeeway(config.Leeway),
		),
	}, nil
}

// Authenticate returns the principal of the token that r carries as "Authorization: Bearer <token>", the
// scheme in any case. The token must be signed with HS256 and the Config's secret, and carry an unexpired
// exp (within the leeway), a sub, and the configured iss and aud. Every other request gets an error that
// wraps auth.ErrUnauthenticated with a short reason; the error never contains the token.
func (v *Verifier) Authenticate(r *http.Request) (auth.Principal, error) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return auth.Principal{}, refusal("no bearer token")
	}

	var tokenClaims claims
	if _, err := v.parser.ParseWithClaims(token, &tokenClaims, v.key); err != nil {
		return auth.Principal{}, refusal(reason(err))
	}
	if tokenClaims.Subject == "" {
		return auth.Principal{}, refusal("token without a subject")
	}

	return auth.Principal{ID: tokenClaims.Subject, Roles: tokenClaims.Roles}, nil
}

func (v *Verifier) key(*jwtlib.Token) (any, error) { return []byte(v.secret.Reveal()), nil }

func refusal(why string) error {
	return fmt.Errorf("%w: %s", auth.ErrUnauthenticated, why)
}

func reason(err error) string {
	switch {
	case errors.Is(err, jwtlib.ErrTokenMalformed):
		return "malformed token"
	case errors.Is(err, jwtlib.ErrTokenExpired):
		return "token expired"
	case errors.Is(err, jwtlib.ErrTokenSignatureInvalid), errors.Is(err, jwtlib.ErrTokenUnverifiable):
		return "invalid signature"
	case errors.Is(err, jwtlib.ErrTokenInvalidIssuer):
		return "token of another issuer"
	case errors.Is(err, jwtlib.ErrTokenInvalidAudience):
		return "token for another audience"
	case errors.Is(err, jwtlib.ErrTokenRequiredClaimMissing):
		return "token without a required claim"
	default:
		return "invalid token"
	}
}
