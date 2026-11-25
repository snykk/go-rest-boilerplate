package jwt_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
	"github.com/snykk/go-rest-boilerplate/internal/platform/auth/jwt"
)

// Issue signs a token for the subject and roles with HS256 and the claims sub, roles, iss, aud, iat and exp,
// returns when it expires, the Config's TTL after now, and refuses a TTL that is not positive and a
// configuration without a secret, an issuer or an audience. A verifier of the same configuration accepts the
// token and gives back the subject and roles as its principal.
func TestIssuerIssue(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	principal := auth.Principal{ID: "u1", Roles: []string{"reader", "editor"}}
	tCases := []struct {
		name    string
		change  func(config *jwt.Config)
		ttl     time.Duration // the Config's TTL
		wantErr bool
	}{
		{
			name:   "a subject gets a token that expires after the ttl",
			change: func(*jwt.Config) {},
			ttl:    15 * time.Minute,
		},
		{name: "a zero ttl is refused", change: func(*jwt.Config) {}, ttl: 0, wantErr: true},
		{name: "a negative ttl is refused", change: func(*jwt.Config) {}, ttl: -time.Minute, wantErr: true},
		{
			name:    "a configuration without a secret is refused",
			change:  func(config *jwt.Config) { config.Secret = nil },
			ttl:     time.Minute,
			wantErr: true,
		},
		{
			name:    "a configuration without an issuer is refused",
			change:  func(config *jwt.Config) { config.Issuer = "" },
			ttl:     time.Minute,
			wantErr: true,
		},
		{
			name:    "a configuration without an audience is refused",
			change:  func(config *jwt.Config) { config.Audience = "" },
			ttl:     time.Minute,
			wantErr: true,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			config := configWith(tc.change)
			config.TTL = tc.ttl
			issuer := jwt.NewIssuer(config, func() time.Time { return now })

			token, expiresAt, err := issuer.Issue(principal.ID, principal.Roles)

			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("Issue error = %v, want error: %t", err, tc.wantErr)
			}
			if tc.wantErr {
				if token != "" || !expiresAt.IsZero() {
					t.Errorf("Issue = %q, %v, want no token and no expiry", token, expiresAt)
				}
				return
			}
			if want := now.Add(tc.ttl); !expiresAt.Equal(want) {
				t.Errorf("Issue expiresAt = %v, want %v", expiresAt, want)
			}
			assertClaims(t, token, config, principal, now, expiresAt)
			assertAccepted(t, token, config, principal)
		})
	}
}

func assertClaims(t *testing.T, token string, config jwt.Config, principal auth.Principal, issued, expires time.Time) {
	t.Helper()
	claims := jwtlib.MapClaims{}
	parsed, _, err := jwtlib.NewParser().ParseUnverified(token, claims)
	if err != nil {
		t.Fatalf("ParseUnverified = %v, want nil", err)
	}
	if algorithm := parsed.Method.Alg(); algorithm != "HS256" {
		t.Errorf("token alg = %q, want HS256", algorithm)
	}
	want := jwtlib.MapClaims{
		"sub":   principal.ID,
		"roles": []any{"reader", "editor"},
		"iss":   config.Issuer,
		"aud":   []any{config.Audience},
		"iat":   float64(issued.Unix()),
		"exp":   float64(expires.Unix()),
	}
	if !reflect.DeepEqual(claims, want) {
		t.Errorf("token claims = %v, want %v", claims, want)
	}
}

func assertAccepted(t *testing.T, token string, config jwt.Config, want auth.Principal) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/articles", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	got, err := jwt.NewVerifier(config).Authenticate(request)

	if err != nil {
		t.Fatalf("Authenticate = %v, want nil", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Authenticate = %+v, want %+v", got, want)
	}
}
