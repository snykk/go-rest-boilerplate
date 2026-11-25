package app_test

import (
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/app"
)

// New refuses a configuration it cannot start with, within seconds.
func TestNew(t *testing.T) {
	const unreachable = "postgres://u:p@127.0.0.1:1/db?connect_timeout=2"
	tCases := []struct {
		name   string
		change func(config *app.Config)
	}{
		{name: "an unreachable database is refused", change: func(config *app.Config) {}},
		{name: "an unknown log format is refused", change: func(config *app.Config) { config.Platform.Log.Format = "xml" }},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := app.LoadConfig(env(map[string]string{
				"DATABASE_URL":        unreachable,
				"IDENTITY_JWT_SECRET": jwtSecret,
			}))
			if err != nil {
				t.Fatalf("LoadConfig = %v", err)
			}
			tc.change(&config)
			start := time.Now()

			_, err = app.New(t.Context(), config)

			if err == nil {
				t.Fatal("New = nil error, want the start-up failure")
			}
			if took := time.Since(start); took > 10*time.Second {
				t.Errorf("New took %v, want under 10s", took)
			}
		})
	}
}

func env(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
}
