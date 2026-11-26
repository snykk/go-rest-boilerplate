package main

import (
	"strings"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
)

// mapSource returns a config.Source that holds values and nothing else, so that no test reads the environment
// of the machine it runs on or its .env file.
func mapSource(values map[string]string) config.Source {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

// loadAccounts reads the credentials of the two accounts from the SEED_* settings, and falls back to the
// development defaults for each one that is unset.
func TestLoadAccounts(t *testing.T) {
	tCases := []struct {
		name string
		env  map[string]string
		want Accounts
	}{
		{
			name: "settings that are not set take the development defaults",
			want: Accounts{
				EditorEmail:    "editor@example.com",
				EditorPassword: "editor-password",
				ReaderEmail:    "reader@example.com",
				ReaderPassword: "reader-password",
			},
		},
		{
			name: "settings that are set override the defaults one by one",
			env: map[string]string{
				"SEED_EDITOR_EMAIL":    "ada@example.com",
				"SEED_READER_PASSWORD": "a-better-password",
			},
			want: Accounts{
				EditorEmail:    "ada@example.com",
				EditorPassword: "editor-password",
				ReaderEmail:    "reader@example.com",
				ReaderPassword: "a-better-password",
			},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got := loadAccounts(mapSource(tc.env))

			if got != tc.want {
				t.Errorf("loadAccounts = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// execute reports the problems of every setting before it opens anything, and refuses to run when APP_ENV is
// production, also before it opens anything: the unreachable DATABASE_URL of that case would otherwise fail
// first. It reads the settings from the source it is given alone.
func TestExecute(t *testing.T) {
	tCases := []struct {
		name            string
		env             map[string]string
		wantErrContains []string
	}{
		{
			name:            "missing settings are reported together",
			wantErrContains: []string{"DATABASE_URL", "IDENTITY_JWT_SECRET"},
		},
		{
			name: "a production environment is refused",
			env: map[string]string{
				"APP_ENV":                   "production",
				"HTTP_CORS_ALLOWED_ORIGINS": "https://example.com",
				"DATABASE_URL":              "postgres://unused",
				"IDENTITY_JWT_SECRET":       "0123456789abcdef0123456789abcdef",
			},
			wantErrContains: []string{"seed data is for development only", "APP_ENV is production"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			err := execute(t.Context(), mapSource(tc.env))

			if err == nil {
				t.Fatalf("execute = nil, want an error holding %q", tc.wantErrContains)
			}
			for _, text := range tc.wantErrContains {
				if !strings.Contains(err.Error(), text) {
					t.Errorf("execute = %v, want an error holding %q", err, text)
				}
			}
		})
	}
}
