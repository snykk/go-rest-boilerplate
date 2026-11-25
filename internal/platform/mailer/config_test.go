package mailer_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
	"github.com/snykk/go-rest-boilerplate/internal/platform/mailer"
)

// defaultsWith returns the configuration that no settings give, changed by change.
func defaultsWith(change func(c *mailer.Config)) mailer.Config {
	settings := mailer.Config{
		Addr:      "localhost:1025",
		From:      "no-reply@example.com",
		Timeout:   10 * time.Second,
		QueueSize: 100,
	}
	if change != nil {
		change(&settings)
	}
	return settings
}

// LoadConfig reads the MAIL_* settings with their defaults and reports every invalid setting in one error.
func TestLoadConfig(t *testing.T) {
	tCases := []struct {
		name         string
		env          map[string]string
		want         mailer.Config
		wantProblems []string
	}{
		{
			name: "no settings give the defaults",
			want: defaultsWith(nil),
		},
		{
			name: "every setting is read",
			env: map[string]string{
				"MAIL_SMTP_ADDR":     "smtp.example.com:587",
				"MAIL_FROM":          "Blog <blog@example.com>",
				"MAIL_SMTP_USERNAME": "mailer",
				"MAIL_SMTP_PASSWORD": "s3cret",
				"MAIL_SMTP_STARTTLS": "true",
				"MAIL_TIMEOUT":       "3s",
				"MAIL_QUEUE_SIZE":    "25",
			},
			want: mailer.Config{
				Addr:      "smtp.example.com:587",
				From:      "Blog <blog@example.com>",
				Username:  "mailer",
				Password:  config.NewSecret("s3cret"),
				StartTLS:  true,
				Timeout:   3 * time.Second,
				QueueSize: 25,
			},
		},
		{
			name:         "a bad timeout and a bad queue size are reported together",
			env:          map[string]string{"MAIL_TIMEOUT": "abc", "MAIL_QUEUE_SIZE": "0"},
			wantProblems: []string{"MAIL_TIMEOUT", "MAIL_QUEUE_SIZE"},
		},
		{
			name:         "a bad boolean is refused",
			env:          map[string]string{"MAIL_SMTP_STARTTLS": "maybe"},
			wantProblems: []string{"MAIL_SMTP_STARTTLS"},
		},
		{
			name:         "an address without a port is refused",
			env:          map[string]string{"MAIL_SMTP_ADDR": "smtp.example.com"},
			wantProblems: []string{"MAIL_SMTP_ADDR"},
		},
		{
			name: "a queue size of 10000 is accepted",
			env:  map[string]string{"MAIL_QUEUE_SIZE": "10000"},
			want: defaultsWith(func(c *mailer.Config) { c.QueueSize = 10_000 }),
		},
		{
			name:         "a queue size above 10000 is refused",
			env:          map[string]string{"MAIL_QUEUE_SIZE": "10001"},
			wantProblems: []string{"MAIL_QUEUE_SIZE"},
		},
		{
			name:         "a sender that is not an email address is refused",
			env:          map[string]string{"MAIL_FROM": "not an address"},
			wantProblems: []string{"MAIL_FROM"},
		},
		{
			name:         "a sender with a quoted local part is refused",
			env:          map[string]string{"MAIL_FROM": `"a b"@example.com`},
			wantProblems: []string{"MAIL_FROM"},
		},
		{
			name: "a sender with a quoted display name is accepted",
			env:  map[string]string{"MAIL_FROM": `"Blog Team" <no-reply@example.com>`},
			want: defaultsWith(func(c *mailer.Config) { c.From = `"Blog Team" <no-reply@example.com>` }),
		},
		{
			name: "a sender outside ASCII is accepted",
			env:  map[string]string{"MAIL_FROM": "josé@example.com"},
			want: defaultsWith(func(c *mailer.Config) { c.From = "josé@example.com" }),
		},
		{
			name: "a sender whose domain is outside ASCII is accepted",
			env:  map[string]string{"MAIL_FROM": "user@bücher.example"},
			want: defaultsWith(func(c *mailer.Config) { c.From = "user@bücher.example" }),
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mailer.LoadConfig(func(key string) (string, bool) { value, ok := tc.env[key]; return value, ok })

			if len(tc.wantProblems) > 0 {
				if err == nil {
					t.Fatalf("LoadConfig = nil error, want one naming %v", tc.wantProblems)
				}
				for _, key := range tc.wantProblems {
					if !strings.Contains(err.Error(), key) {
						t.Errorf("LoadConfig error = %q, want it to name %s", err, key)
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
