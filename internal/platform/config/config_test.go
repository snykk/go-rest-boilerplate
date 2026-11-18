package config_test

import (
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
)

func env(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
}

// defaults is the Config FromEnv builds when only DATABASE_URL=postgres://x is set, changed by change.
func defaults(change func(*config.Config)) config.Config {
	want := config.Config{
		Env:     "development",
		Service: "go-rest-boilerplate",
		HTTP: config.HTTP{
			Addr:               ":8080",
			ReadHeaderTimeout:  5 * time.Second,
			ReadTimeout:        15 * time.Second,
			WriteTimeout:       15 * time.Second,
			IdleTimeout:        60 * time.Second,
			ShutdownTimeout:    10 * time.Second,
			BodyLimitBytes:     1 << 20,
			CORSAllowedOrigins: []string{"*"},
		},
		Log:      config.Log{Level: slog.LevelInfo, Format: "text", Source: true},
		Database: config.Database{URL: "postgres://x"},
		Swagger:  true,
	}
	if change != nil {
		change(&want)
	}
	return want
}

// FromEnv fills every setting from the environment or its default, and reports every invalid or missing
// setting in one error.
func TestFromEnv(t *testing.T) {
	tCases := []struct {
		name    string
		env     map[string]string
		want    config.Config
		wantErr []string // keys the error must name; nil means success
	}{
		{
			name: "an environment with only the database URL gets every default",
			env:  map[string]string{"DATABASE_URL": "postgres://x"},
			want: defaults(nil),
		},
		{
			name: "production switches to JSON logs, turns Swagger off and takes the listed origins",
			env: map[string]string{
				"APP_ENV":                   "production",
				"DATABASE_URL":              "postgres://x",
				"HTTP_CORS_ALLOWED_ORIGINS": " https://a.example , https://b.example ",
			},
			want: defaults(func(want *config.Config) {
				want.Env, want.Log.Format, want.Swagger = "production", "json", false
				want.HTTP.CORSAllowedOrigins = []string{"https://a.example", "https://b.example"}
			}),
		},
		{
			name: "trusted proxies are read as CIDR prefixes",
			env: map[string]string{
				"DATABASE_URL":         "postgres://x",
				"HTTP_TRUSTED_PROXIES": "10.0.0.0/8, 192.168.1.10/32",
			},
			want: defaults(func(want *config.Config) {
				want.HTTP.TrustedProxies = []netip.Prefix{
					netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("192.168.1.10/32"),
				}
			}),
		},
		{
			name: "the service name and caller logging are read",
			env: map[string]string{
				"DATABASE_URL": "postgres://x",
				"SERVICE_NAME": "articles",
				"LOG_SOURCE":   "false",
			},
			want: defaults(func(want *config.Config) {
				want.Service, want.Log.Source = "articles", false
			}),
		},
		{
			name: "every invalid or missing setting is reported at once",
			env: map[string]string{
				"APP_ENV":              "staging",
				"HTTP_READ_TIMEOUT":    "abc",
				"HTTP_TRUSTED_PROXIES": "foo",
			},
			wantErr: []string{"APP_ENV", "HTTP_READ_TIMEOUT", "HTTP_TRUSTED_PROXIES", "DATABASE_URL"},
		},
		{
			name:    "production refuses to start without CORS origins",
			env:     map[string]string{"APP_ENV": "production", "DATABASE_URL": "postgres://x"},
			wantErr: []string{"HTTP_CORS_ALLOWED_ORIGINS"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := config.FromEnv(env(tc.env))

			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("FromEnv = %+v, want an error naming %v", got, tc.wantErr)
				}
				for _, key := range tc.wantErr {
					if !strings.Contains(err.Error(), key) {
						t.Errorf("FromEnv error = %q, want it to name %s", err, key)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("FromEnv = %v, want no error", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("FromEnv = %+v, want %+v", got, tc.want)
			}
			if want := tc.want.Env == "production"; got.Production() != want {
				t.Errorf("Production() = %t, want %t", got.Production(), want)
			}
		})
	}
}

// LoadSource reads an optional .env file and lets the process environment win over it, without writing the
// file's values into the process environment.
func TestLoadSource(t *testing.T) {
	tCases := []struct {
		name     string
		dotenv   string            // contents of ./.env; "" means no file
		env      map[string]string // process environment
		wantURL  string
		wantAddr string
	}{
		{
			name:    "settings come from the environment alone when there is no .env file",
			env:     map[string]string{"DATABASE_URL": "postgres://from-env"},
			wantURL: "postgres://from-env",
		},
		{
			name:    "a .env file alone is enough",
			dotenv:  "DATABASE_URL=postgres://from-file\n",
			wantURL: "postgres://from-file",
		},
		{
			name:     "the process environment wins over the .env file",
			dotenv:   "DATABASE_URL=postgres://from-file\nHTTP_ADDR=:9999\n",
			env:      map[string]string{"DATABASE_URL": "postgres://from-env"},
			wantURL:  "postgres://from-env",
			wantAddr: ":9999",
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			if tc.dotenv != "" {
				if err := os.WriteFile(filepath.Join(directory, ".env"), []byte(tc.dotenv), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			t.Chdir(directory)
			for _, key := range []string{"DATABASE_URL", "HTTP_ADDR"} {
				t.Setenv(key, "") // empty counts as unset; isolates the case from the developer's shell
			}
			for key, value := range tc.env {
				t.Setenv(key, value)
			}

			source, err := config.LoadSource()

			if err != nil {
				t.Fatalf("LoadSource = %v, want no error", err)
			}
			url, _ := source("DATABASE_URL")
			address, _ := source("HTTP_ADDR")
			if url != tc.wantURL || address != tc.wantAddr {
				t.Errorf("LoadSource = URL %q, addr %q; want %q, %q", url, address, tc.wantURL, tc.wantAddr)
			}
			if value := os.Getenv("HTTP_ADDR"); value != "" {
				t.Errorf("HTTP_ADDR = %q after LoadSource, want the .env value kept out of the process environment", value)
			}
		})
	}
}

// LoadDatabase reads only DATABASE_URL, so settings the API needs cannot stop a migration.
func TestLoadDatabase(t *testing.T) {
	tCases := []struct {
		name    string
		env     map[string]string
		wantURL string
		wantErr bool
	}{
		{
			name:    "production settings the API would refuse do not matter",
			env:     map[string]string{"APP_ENV": "production", "DATABASE_URL": "postgres://x"},
			wantURL: "postgres://x",
		},
		{name: "a missing database URL is refused", wantErr: true},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("DATABASE_URL", "")
			for key, value := range tc.env {
				t.Setenv(key, value)
			}

			database, err := config.LoadDatabase()

			if (err != nil) != tc.wantErr {
				t.Fatalf("LoadDatabase = %v, want an error: %t", err, tc.wantErr)
			}
			if database.URL != tc.wantURL {
				t.Errorf("LoadDatabase URL = %q, want %q", database.URL, tc.wantURL)
			}
		})
	}
}
