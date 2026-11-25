package jwt_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
	"github.com/snykk/go-rest-boilerplate/internal/platform/auth/jwt"
)

func configFixture() jwt.Config {
	return jwt.Config{
		Secret:   []byte(strings.Repeat("s", 32)),
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
	token, _, err := jwt.NewIssuer(config, func() time.Time { return now }).Issue("u1", []string{"reader"})
	if err != nil {
		t.Fatalf("Issue = %v, want nil", err)
	}
	return token
}

// Authenticate identifies the caller of a request that carries a bearer token the issuer signed, and refuses,
// with an error that wraps auth.ErrUnauthenticated and never repeats the token, every request without one: no
// header, another scheme, an expired token, a token of another issuer, audience, secret or algorithm, a token
// without an expiry or a subject, and garbage.
func TestVerifierAuthenticate(t *testing.T) {
	config := configFixture()
	reader := auth.Principal{ID: "u1", Roles: []string{"reader"}}
	now := time.Now()
	valid := issuedAt(t, config, now, time.Hour)
	otherIssuer := configWith(func(other *jwt.Config) { other.Issuer = "other" })
	otherAudience := configWith(func(other *jwt.Config) { other.Audience = "other" })
	otherSecret := configWith(func(other *jwt.Config) { other.Secret = []byte(strings.Repeat("o", 32)) })
	tCases := []struct {
		name          string
		authorization string
		want          auth.Principal
		wantErr       error
	}{
		{name: "a token the issuer signed gives its principal", authorization: "Bearer " + valid, want: reader},
		{name: "the scheme is matched in any case", authorization: "bearer " + valid, want: reader},
		{
			name: "a token without roles gives a principal without roles",
			authorization: "Bearer " + handSigned(t, jwtlib.SigningMethodHS256, config.Secret,
				claimsWith(func(claims jwtlib.MapClaims) { delete(claims, "roles") })),
			want: auth.Principal{ID: "u1"},
		},
		{name: "no Authorization header is unauthenticated", wantErr: auth.ErrUnauthenticated},
		{name: "another scheme is unauthenticated", authorization: "Basic " + valid, wantErr: auth.ErrUnauthenticated},
		{
			name:          "a scheme without a token is unauthenticated",
			authorization: "Bearer",
			wantErr:       auth.ErrUnauthenticated,
		},
		{name: "an empty token is unauthenticated", authorization: "Bearer ", wantErr: auth.ErrUnauthenticated},
		{
			name:          "an expired token is unauthenticated",
			authorization: "Bearer " + issuedAt(t, config, now.Add(-2*time.Hour), time.Hour),
			wantErr:       auth.ErrUnauthenticated,
		},
		{
			name:          "an expired token within the leeway is accepted",
			authorization: "Bearer " + issuedAt(t, config, now.Add(-time.Hour-10*time.Second), time.Hour),
			want:          reader,
		},
		{
			name:          "a token of another issuer is unauthenticated",
			authorization: "Bearer " + issuedAt(t, otherIssuer, now, time.Hour),
			wantErr:       auth.ErrUnauthenticated,
		},
		{
			name:          "a token for another audience is unauthenticated",
			authorization: "Bearer " + issuedAt(t, otherAudience, now, time.Hour),
			wantErr:       auth.ErrUnauthenticated,
		},
		{
			name: "a token without an issuer is unauthenticated",
			authorization: "Bearer " + handSigned(t, jwtlib.SigningMethodHS256, config.Secret,
				claimsWith(func(claims jwtlib.MapClaims) { delete(claims, "iss") })),
			wantErr: auth.ErrUnauthenticated,
		},
		{
			name: "a token without an audience is unauthenticated",
			authorization: "Bearer " + handSigned(t, jwtlib.SigningMethodHS256, config.Secret,
				claimsWith(func(claims jwtlib.MapClaims) { delete(claims, "aud") })),
			wantErr: auth.ErrUnauthenticated,
		},
		{
			name:          "a token signed with another secret is unauthenticated",
			authorization: "Bearer " + issuedAt(t, otherSecret, now, time.Hour),
			wantErr:       auth.ErrUnauthenticated,
		},
		{
			name: "a token with alg none is unauthenticated",
			authorization: "Bearer " + handSigned(t, jwtlib.SigningMethodNone, jwtlib.UnsafeAllowNoneSignatureType,
				claimsFixture()),
			wantErr: auth.ErrUnauthenticated,
		},
		{
			name:          "a token of another HMAC algorithm is unauthenticated",
			authorization: "Bearer " + handSigned(t, jwtlib.SigningMethodHS384, config.Secret, claimsFixture()),
			wantErr:       auth.ErrUnauthenticated,
		},
		{
			name: "a token without exp is unauthenticated",
			authorization: "Bearer " + handSigned(t, jwtlib.SigningMethodHS256, config.Secret,
				claimsWith(func(claims jwtlib.MapClaims) { delete(claims, "exp") })),
			wantErr: auth.ErrUnauthenticated,
		},
		{
			name: "a token without sub is unauthenticated",
			authorization: "Bearer " + handSigned(t, jwtlib.SigningMethodHS256, config.Secret,
				claimsWith(func(claims jwtlib.MapClaims) { delete(claims, "sub") })),
			wantErr: auth.ErrUnauthenticated,
		},
		{
			name:          "a token without its signature is unauthenticated",
			authorization: "Bearer " + valid[:strings.LastIndex(valid, ".")+1],
			wantErr:       auth.ErrUnauthenticated,
		},
		{name: "garbage is unauthenticated", authorization: "Bearer not.a.token", wantErr: auth.ErrUnauthenticated},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/articles", nil)
			if tc.authorization != "" {
				request.Header.Set("Authorization", tc.authorization)
			}

			got, err := jwt.NewVerifier(config).Authenticate(request)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Authenticate = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Authenticate = %+v, want %+v", got, tc.want)
			}
			if _, token, ok := strings.Cut(tc.authorization, " "); err != nil && ok && token != "" &&
				strings.Contains(err.Error(), token) {
				t.Errorf("Authenticate error = %q, want it without the token", err)
			}
		})
	}
}

// Authenticate refuses every request when the configuration lacks a secret, an issuer or an audience, rather
// than accept tokens it cannot check.
func TestVerifierAuthenticateIncompleteConfig(t *testing.T) {
	tCases := []struct {
		name   string
		change func(config *jwt.Config)
		claims jwtlib.MapClaims
	}{
		{name: "without a secret", change: func(config *jwt.Config) { config.Secret = nil }, claims: claimsFixture()},
		{
			name:   "without an issuer",
			change: func(config *jwt.Config) { config.Issuer = "" },
			claims: claimsWith(func(claims jwtlib.MapClaims) { delete(claims, "iss") }),
		},
		{
			name:   "without an audience",
			change: func(config *jwt.Config) { config.Audience = "" },
			claims: claimsWith(func(claims jwtlib.MapClaims) { delete(claims, "aud") }),
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			signedWith := configFixture().Secret
			request := httptest.NewRequest(http.MethodGet, "/v1/articles", nil)
			request.Header.Set("Authorization", "Bearer "+handSigned(t, jwtlib.SigningMethodHS256, signedWith, tc.claims))

			_, err := jwt.NewVerifier(configWith(tc.change)).Authenticate(request)

			if !errors.Is(err, auth.ErrUnauthenticated) {
				t.Errorf("Authenticate = %v, want %v", err, auth.ErrUnauthenticated)
			}
		})
	}
}
