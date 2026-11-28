package jwt_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/snykk/go-rest-boilerplate/internal/platform/auth/jwt"
	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
)

func configFixture() jwt.Config {
	return jwt.Config{
		Secret:   config.NewSecret(strings.Repeat("s", 32)),
		Issuer:   "blog",
		Audience: "blog-api",
		Leeway:   30 * time.Second,
	}
}

func configWith(change func(config *jwt.Config)) jwt.Config {
	config := configFixture()
	change(&config)
	return config
}

func claimsFixture() jwtlib.MapClaims {
	now := time.Now()
	return jwtlib.MapClaims{
		"sub":   "u1",
		"roles": []string{"reader"},
		"iss":   "blog",
		"aud":   "blog-api",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
}

func claimsWith(change func(claims jwtlib.MapClaims)) jwtlib.MapClaims {
	claims := claimsFixture()
	change(claims)
	return claims
}

func handSigned(t *testing.T, method jwtlib.SigningMethod, key any, claims jwtlib.MapClaims) string {
	t.Helper()
	token, err := jwtlib.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("SignedString = %v, want nil", err)
	}
	return token
}

func issuedAt(t *testing.T, config jwt.Config, now time.Time, ttl time.Duration) string {
	t.Helper()
	config.TTL = ttl
	issuer, err := jwt.NewIssuer(config, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewIssuer = %v, want nil", err)
	}
	token, _, err := issuer.Issue("u1", []string{"reader"})
	if err != nil {
		t.Fatalf("Issue = %v, want nil", err)
	}
	return token
}

func newVerifier(t *testing.T, config jwt.Config) *jwt.Verifier {
	t.Helper()
	verifier, err := jwt.NewVerifier(config)
	if err != nil {
		t.Fatalf("NewVerifier = %v, want nil", err)
	}
	return verifier
}

func printsSecret(printed, secret string) bool {
	forms := []string{secret, fmt.Sprintf("%v", []byte(secret)), fmt.Sprintf("%#v", []byte(secret))}
	for _, form := range forms {
		if strings.Contains(printed, form) {
			return true
		}
	}

	return false
}
