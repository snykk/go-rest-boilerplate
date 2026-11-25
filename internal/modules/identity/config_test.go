package identity_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity"
	"github.com/snykk/go-rest-boilerplate/internal/platform/auth/jwt"
)

// secret is a JWT secret of exactly 32 bytes, the shortest LoadConfig accepts.
const secret = "0123456789abcdef0123456789abcdef"

// configFixture is the configuration LoadConfig reads from secret and the defaults.
func configFixture() identity.Config {
	return identity.Config{
		JWTSecret:         secret,
		JWTIssuer:         "go-rest-boilerplate",
		JWTAudience:       "go-rest-boilerplate",
		AccessTTL:         15 * time.Minute,
		RefreshTTL:        720 * time.Hour,
		RefreshReuseGrace: 10 * time.Second,
		CodeTTL:           10 * time.Minute,
		CodeAttempts:      5,
		ResetTTL:          30 * time.Minute,
		ResendCooldown:    time.Minute,
		BcryptCost:        12,
		LoginAttempts:     5,
		Lockout:           15 * time.Minute,
	}
}

// LoadConfig reads the identity settings from IDENTITY_* variables, with defaults for all but the secret. It
// requires a JWT secret of at least 32 bytes, a bcrypt cost from 10 to 14 and lifetimes of at least a minute,
// and reports every problem in one error.
func TestLoadConfig(t *testing.T) {
	tCases := []struct {
		name         string
		env          map[string]string
		want         identity.Config
		wantProblems []string // the keys the error must name; none means no error
	}{
		{
			name: "a secret alone gives the defaults",
			env:  map[string]string{"IDENTITY_JWT_SECRET": secret},
			want: configFixture(),
		},
		{
			name: "set values are used",
			env: map[string]string{
				"IDENTITY_JWT_SECRET":          secret + "-longer",
				"IDENTITY_JWT_ISSUER":          "blog",
				"IDENTITY_JWT_AUDIENCE":        "blog-api",
				"IDENTITY_ACCESS_TTL":          "5m",
				"IDENTITY_REFRESH_TTL":         "24h",
				"IDENTITY_REFRESH_REUSE_GRACE": "3s",
				"IDENTITY_CODE_TTL":            "1m",
				"IDENTITY_CODE_ATTEMPTS":       "3",
				"IDENTITY_RESET_TTL":           "1h",
				"IDENTITY_RESEND_COOLDOWN":     "2m",
				"IDENTITY_BCRYPT_COST":         "10",
				"IDENTITY_LOGIN_ATTEMPTS":      "10",
				"IDENTITY_LOCKOUT":             "1h",
			},
			want: identity.Config{
				JWTSecret:         secret + "-longer",
				JWTIssuer:         "blog",
				JWTAudience:       "blog-api",
				AccessTTL:         5 * time.Minute,
				RefreshTTL:        24 * time.Hour,
				RefreshReuseGrace: 3 * time.Second,
				CodeTTL:           time.Minute,
				CodeAttempts:      3,
				ResetTTL:          time.Hour,
				ResendCooldown:    2 * time.Minute,
				BcryptCost:        10,
				LoginAttempts:     10,
				Lockout:           time.Hour,
			},
		},
		{name: "a missing secret is refused", wantProblems: []string{"IDENTITY_JWT_SECRET"}},
		{
			name:         "a 31-byte secret and a bcrypt cost of 9 are reported together",
			env:          map[string]string{"IDENTITY_JWT_SECRET": secret[:31], "IDENTITY_BCRYPT_COST": "9"},
			wantProblems: []string{"IDENTITY_JWT_SECRET", "IDENTITY_BCRYPT_COST"},
		},
		{
			name:         "a bcrypt cost above 14 is refused",
			env:          map[string]string{"IDENTITY_JWT_SECRET": secret, "IDENTITY_BCRYPT_COST": "15"},
			wantProblems: []string{"IDENTITY_BCRYPT_COST"},
		},
		{
			name: "lifetimes under a minute are refused",
			env: map[string]string{
				"IDENTITY_JWT_SECRET":  secret,
				"IDENTITY_ACCESS_TTL":  "59s",
				"IDENTITY_REFRESH_TTL": "30s",
				"IDENTITY_CODE_TTL":    "10s",
				"IDENTITY_RESET_TTL":   "1s",
			},
			wantProblems: []string{
				"IDENTITY_ACCESS_TTL",
				"IDENTITY_REFRESH_TTL",
				"IDENTITY_CODE_TTL",
				"IDENTITY_RESET_TTL",
			},
		},
		{
			name: "counts and durations that are not positive are refused",
			env: map[string]string{
				"IDENTITY_JWT_SECRET":          secret,
				"IDENTITY_CODE_ATTEMPTS":       "0",
				"IDENTITY_LOGIN_ATTEMPTS":      "many",
				"IDENTITY_REFRESH_REUSE_GRACE": "0s",
				"IDENTITY_LOCKOUT":             "-1m",
				"IDENTITY_RESEND_COOLDOWN":     "0s",
			},
			wantProblems: []string{
				"IDENTITY_CODE_ATTEMPTS",
				"IDENTITY_LOGIN_ATTEMPTS",
				"IDENTITY_REFRESH_REUSE_GRACE",
				"IDENTITY_LOCKOUT",
				"IDENTITY_RESEND_COOLDOWN",
			},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := identity.LoadConfig(func(key string) (string, bool) { value, ok := tc.env[key]; return value, ok })

			if len(tc.wantProblems) > 0 {
				for _, key := range tc.wantProblems {
					if err == nil || !strings.Contains(err.Error(), key) {
						t.Errorf("LoadConfig = %v, want an error naming %s", err, key)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadConfig = %v, want no error", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("LoadConfig = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// JWT returns the settings the access token's issuer and verifier share: the secret, issuer and audience, the
// access token's lifetime, and a leeway of 10 seconds for clock drift between replicas.
func TestConfigJWT(t *testing.T) {
	want := jwt.Config{
		Secret:   []byte(secret),
		Issuer:   "go-rest-boilerplate",
		Audience: "go-rest-boilerplate",
		TTL:      15 * time.Minute,
		Leeway:   10 * time.Second,
	}

	got := configFixture().JWT()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("JWT = %+v, want %+v", got, want)
	}
}
