package app_test

import (
	"strings"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/app"
)

// LoadConfig reads the platform's settings and every module's from one source, and reports the problems of
// all of them in one error.
func TestLoadConfig(t *testing.T) {
	tCases := []struct {
		name         string
		env          map[string]string
		wantProblems []string // the keys the error must name; none means no error
	}{
		{
			name: "valid settings give the platform's and the modules' configuration",
			env:  map[string]string{"DATABASE_URL": "postgres://x", "ARTICLE_MAX_PAGE_SIZE": "50"},
		},
		{
			name:         "problems of the platform and of a module are reported together",
			env:          map[string]string{"ARTICLE_MAX_PAGE_SIZE": "0"},
			wantProblems: []string{"DATABASE_URL", "ARTICLE_MAX_PAGE_SIZE"},
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
			if got.Platform.Database.URL.Reveal() != "postgres://x" || got.Article.MaxPageSize != 50 {
				t.Errorf("LoadConfig = URL %q, article max page size %d; want %q, 50",
					got.Platform.Database.URL.Reveal(), got.Article.MaxPageSize, "postgres://x")
			}
		})
	}
}
