package jwt_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
	"github.com/snykk/go-rest-boilerplate/internal/platform/auth/jwt"
	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
)

// Authenticate identifies the caller of a request that carries a bearer token the issuer signed, and refuses,
// with an error that wraps auth.ErrUnauthenticated and never repeats the token, every request without one: no
// header, another scheme, an expired token, a token of another issuer, audience, secret or algorithm, a token
// without an expiry or a subject, and garbage.
func TestVerifierAuthenticate(t *testing.T) {
	otherSecret := config.NewSecret(strings.Repeat("o", 32))
	config := configFixture()
	key := []byte(config.Secret.Reveal())
	reader := auth.Principal{ID: "u1", Roles: []string{"reader"}}
	now := time.Now()
	valid := issuedAt(t, config, now, time.Hour)
	otherIssuer := configWith(func(other *jwt.Config) { other.Issuer = "other" })
	otherAudience := configWith(func(other *jwt.Config) { other.Audience = "other" })
	otherSecretConfig := configWith(func(other *jwt.Config) { other.Secret = otherSecret })
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
			authorization: "Bearer " + handSigned(t, jwtlib.SigningMethodHS256, key,
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
			authorization: "Bearer " + handSigned(t, jwtlib.SigningMethodHS256, key,
				claimsWith(func(claims jwtlib.MapClaims) { delete(claims, "iss") })),
			wantErr: auth.ErrUnauthenticated,
		},
		{
			name: "a token without an audience is unauthenticated",
			authorization: "Bearer " + handSigned(t, jwtlib.SigningMethodHS256, key,
				claimsWith(func(claims jwtlib.MapClaims) { delete(claims, "aud") })),
			wantErr: auth.ErrUnauthenticated,
		},
		{
			name:          "a token signed with another secret is unauthenticated",
			authorization: "Bearer " + issuedAt(t, otherSecretConfig, now, time.Hour),
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
			authorization: "Bearer " + handSigned(t, jwtlib.SigningMethodHS384, key, claimsFixture()),
			wantErr:       auth.ErrUnauthenticated,
		},
		{
			name: "a token without exp is unauthenticated",
			authorization: "Bearer " + handSigned(t, jwtlib.SigningMethodHS256, key,
				claimsWith(func(claims jwtlib.MapClaims) { delete(claims, "exp") })),
			wantErr: auth.ErrUnauthenticated,
		},
		{
			name: "a token without sub is unauthenticated",
			authorization: "Bearer " + handSigned(t, jwtlib.SigningMethodHS256, key,
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

			got, err := newVerifier(t, config).Authenticate(request)

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

// NewVerifier returns a verifier for a configuration with a secret of at least 32 bytes, an issuer and an
// audience, and refuses every other without repeating the secret. A verifier printed whole never shows its
// secret.
func TestNewVerifier(t *testing.T) {
	secretText := strings.Repeat("s", 32)
	tCases := []struct {
		name    string
		change  func(config *jwt.Config)
		wantErr bool
	}{
		{name: "a complete configuration gives a verifier that never prints its secret", change: func(*jwt.Config) {}},
		{
			name:    "a secret of 31 bytes is refused",
			change:  func(c *jwt.Config) { c.Secret = config.NewSecret(secretText[:31]) },
			wantErr: true,
		},
		{
			name:    "an empty issuer is refused",
			change:  func(config *jwt.Config) { config.Issuer = "" },
			wantErr: true,
		},
		{
			name:    "an empty audience is refused",
			change:  func(config *jwt.Config) { config.Audience = "" },
			wantErr: true,
		},
		{
			name:    "a negative leeway is refused",
			change:  func(config *jwt.Config) { config.Leeway = -time.Second },
			wantErr: true,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := configWith(tc.change)

			verifier, err := jwt.NewVerifier(cfg)

			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("NewVerifier = %v, want error: %t", err, tc.wantErr)
			}
			if tc.wantErr {
				if verifier != nil {
					t.Errorf("NewVerifier = %v, want no verifier", verifier)
				}
				if strings.Contains(err.Error(), cfg.Secret.Reveal()) {
					t.Errorf("NewVerifier error = %q, want it without the secret", err)
				}
				return
			}
			if printed := fmt.Sprintf("%+v %#v", verifier, verifier); printsSecret(printed, secretText) {
				t.Errorf("printed verifier = %q, want it without the secret", printed)
			}
		})
	}
}
