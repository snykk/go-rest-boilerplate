package database_test

import (
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/platform/database"
)

// Open refuses a URL it cannot parse and a server it cannot reach, without hanging.
func TestOpen(t *testing.T) {
	tCases := []struct {
		name, url string
	}{
		{name: "an unparsable URL is refused", url: "postgres://%zz"},
		{name: "an unreachable server is refused quickly", url: "postgres://u:p@127.0.0.1:1/db?connect_timeout=2"},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()

			_, err := database.Open(t.Context(), tc.url)

			if err == nil {
				t.Fatal("Open = nil, want an error")
			}
			if took := time.Since(start); took > 10*time.Second {
				t.Errorf("Open took %v, want under 10s", took)
			}
		})
	}
}
