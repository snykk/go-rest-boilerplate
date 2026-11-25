package jwt_test

import (
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

// NewIssuer returns an issuer for a configuration with a secret of at least 32 bytes, an issuer, an audience
// and a positive ttl, and a clock, and refuses every other without repeating the secret. An issuer printed
// whole never shows its secret.
func TestNewIssuer(t *testing.T) {
	secretText := strings.Repeat("s", 32)
	clock := time.Now
	tCases := []struct {
		name    string
		change  func(config *jwt.Config)
		clock   func() time.Time
		wantErr bool
	}{
		{name: "a complete configuration gives an issuer that never prints its secret", clock: clock},
		{
			name:    "a secret of 31 bytes is refused",
			change:  func(c *jwt.Config) { c.Secret = config.NewSecret(secretText[:31]) },
			clock:   clock,
			wantErr: true,
		},
		{name: "a secret of 32 bytes is accepted", clock: clock},
		{
			name:    "an empty issuer is refused",
			change:  func(c *jwt.Config) { c.Issuer = "" },
			clock:   clock,
			wantErr: true,
		},
		{
			name:    "an empty audience is refused",
			change:  func(c *jwt.Config) { c.Audience = "" },
			clock:   clock,
			wantErr: true,
		},
		{
			name:    "a zero ttl is refused",
			change:  func(c *jwt.Config) { c.TTL = 0 },
			clock:   clock,
			wantErr: true,
		},
		{
			name:    "a negative ttl is refused",
			change:  func(c *jwt.Config) { c.TTL = -time.Minute },
			clock:   clock,
			wantErr: true,
		},
		{name: "a nil clock is refused", clock: nil, wantErr: true},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := configWith(func(c *jwt.Config) {
				c.TTL = time.Minute
				if tc.change != nil {
					tc.change(c)
				}
			})

			issuer, err := jwt.NewIssuer(cfg, tc.clock)

			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("NewIssuer = %v, want error: %t", err, tc.wantErr)
			}
			if tc.wantErr {
				if issuer != nil {
					t.Errorf("NewIssuer = %v, want no issuer", issuer)
				}
				if strings.Contains(err.Error(), cfg.Secret.Reveal()) {
					t.Errorf("NewIssuer error = %q, want it without the secret", err)
				}
				return
			}
			if printed := fmt.Sprintf("%+v %#v", issuer, issuer); printsSecret(printed, secretText) {
				t.Errorf("printed issuer = %q, want it without the secret", printed)
			}
		})
	}
}

// Issue signs a token for the subject and roles with HS256 and the claims sub, roles, iss, aud, iat and exp,
// the times in whole seconds, and returns when it expires, the value of exp. It refuses an empty subject. A
// verifier of the same configuration accepts the token and gives back the subject and roles as its principal.
func TestIssuerIssue(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second).Add(200 * time.Millisecond)
	tCases := []struct {
		name    string
		subject string
		ttl     time.Duration
		wantErr bool
	}{
		{name: "a subject gets a token that expires after the ttl", subject: "u1", ttl: 15 * time.Minute},
		{name: "an empty subject is refused", subject: "", ttl: 15 * time.Minute, wantErr: true},
		{name: "a ttl that is not whole seconds expires at the exp claim", subject: "u1", ttl: 90500 * time.Millisecond},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := configFixture()
			cfg.TTL = tc.ttl
			issuer, err := jwt.NewIssuer(cfg, func() time.Time { return now })
			if err != nil {
				t.Fatalf("NewIssuer = %v, want nil", err)
			}
			roles := []string{"reader", "editor"}

			token, expiresAt, err := issuer.Issue(tc.subject, roles)

			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("Issue = %v, want error: %t", err, tc.wantErr)
			}
			if tc.wantErr {
				if token != "" || !expiresAt.IsZero() {
					t.Errorf("Issue = %q, %v, want no token and no expiry", token, expiresAt)
				}
				return
			}
			if want := now.Add(tc.ttl).Truncate(time.Second); !expiresAt.Equal(want) {
				t.Errorf("Issue expiresAt = %v, want %v", expiresAt, want)
			}
			principal := auth.Principal{ID: tc.subject, Roles: roles}
			exp := assertClaims(t, token, cfg, principal, now.Truncate(time.Second))
			if !expiresAt.Equal(time.Unix(exp, 0)) {
				t.Errorf("Issue expiresAt = %v, want the exp claim %v", expiresAt, time.Unix(exp, 0))
			}
			assertAccepted(t, token, cfg, principal)
		})
	}
}

func assertClaims(
	t *testing.T, token string, config jwt.Config, principal auth.Principal, issued time.Time,
) (exp int64) {
	t.Helper()
	claims := jwtlib.MapClaims{}
	parsed, _, err := jwtlib.NewParser().ParseUnverified(token, claims)
	if err != nil {
		t.Fatalf("ParseUnverified = %v, want nil", err)
	}
	if algorithm := parsed.Method.Alg(); algorithm != "HS256" {
		t.Errorf("token alg = %q, want HS256", algorithm)
	}
	expClaim, ok := claims["exp"].(float64)
	if !ok {
		t.Fatalf("token exp = %v, want a number", claims["exp"])
	}
	want := jwtlib.MapClaims{
		"sub":   principal.ID,
		"roles": []any{"reader", "editor"},
		"iss":   config.Issuer,
		"aud":   []any{config.Audience},
		"iat":   float64(issued.Unix()),
		"exp":   expClaim,
	}
	if !reflect.DeepEqual(claims, want) {
		t.Errorf("token claims = %v, want %v", claims, want)
	}
	return int64(expClaim)
}

func assertAccepted(t *testing.T, token string, config jwt.Config, want auth.Principal) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/articles", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	got, err := newVerifier(t, config).Authenticate(request)

	if err != nil {
		t.Fatalf("Authenticate = %v, want nil", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Authenticate = %+v, want %+v", got, want)
	}
}
