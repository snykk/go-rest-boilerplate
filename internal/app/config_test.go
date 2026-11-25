package app_test

import (
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/app"
)

// jwtSecret is a JWT secret of the 32 bytes the identity module requires at least.
const jwtSecret = "0123456789abcdef0123456789abcdef"

// LoadConfig reads the platform's settings, the mailer's and every module's from one source, and reports the
// problems of all of them in one error. It refuses the published development JWT secret in production.
func TestLoadConfig(t *testing.T) {
	tCases := []struct {
		name         string
		env          map[string]string
		wantProblems []string // the keys the error must name; none means no error
	}{
		{
			name: "valid settings give the platform's, the mailer's and the modules' configuration",
			env: map[string]string{
				"DATABASE_URL":          "postgres://x",
				"MAIL_SMTP_ADDR":        "mail:25",
				"ARTICLE_MAX_PAGE_SIZE": "50",
				"IDENTITY_JWT_SECRET":   jwtSecret,
				"IDENTITY_ACCESS_TTL":   "5m",
			},
		},
		{
			name: "problems of the platform, the mailer and the modules are reported together",
			env: map[string]string{
				"ARTICLE_MAX_PAGE_SIZE": "0",
				"MAIL_FROM":             "nobody",
			},
			wantProblems: []string{"DATABASE_URL", "MAIL_FROM", "ARTICLE_MAX_PAGE_SIZE", "IDENTITY_JWT_SECRET"},
		},
		{
			name: "the development JWT secret of .env.example is refused in production",
			env: map[string]string{
				"APP_ENV":                   "production",
				"DATABASE_URL":              "postgres://x",
				"HTTP_CORS_ALLOWED_ORIGINS": "https://blog.example.com",
				"IDENTITY_JWT_SECRET":       "development-secret-32-bytes-long",
			},
			wantProblems: []string{"IDENTITY_JWT_SECRET"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := app.LoadConfig(env(tc.env))

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
			if got.Platform.Database.URL != "postgres://x" || got.Mail.Addr != "mail:25" ||
				got.Article.MaxPageSize != 50 || got.Identity.AccessTTL != 5*time.Minute {
				t.Errorf("LoadConfig = URL %q, mail %q, article max page size %d, access TTL %v; want %q, %q, 50, 5m",
					got.Platform.Database.URL, got.Mail.Addr, got.Article.MaxPageSize, got.Identity.AccessTTL,
					"postgres://x", "mail:25")
			}
		})
	}
}
