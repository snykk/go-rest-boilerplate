package article_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article"
)

// LoadConfig reads the page sizes from ARTICLE_* settings, with defaults, and refuses a default page size
// above the maximum.
func TestLoadConfig(t *testing.T) {
	tCases := []struct {
		name        string
		env         map[string]string
		want        article.Config
		wantProblem string // the key the error must name; "" means no error
	}{
		{name: "no settings give the defaults", want: article.Config{DefaultPageSize: 20, MaxPageSize: 100}},
		{
			name: "set page sizes are used",
			env: map[string]string{
				"ARTICLE_DEFAULT_PAGE_SIZE": "10",
				"ARTICLE_MAX_PAGE_SIZE":     "50",
			},
			want: article.Config{
				DefaultPageSize: 10,
				MaxPageSize:     50,
			},
		},
		{
			name:        "a page size that is not a positive integer is refused",
			env:         map[string]string{"ARTICLE_MAX_PAGE_SIZE": "0"},
			wantProblem: "ARTICLE_MAX_PAGE_SIZE",
		},
		{
			name:        "a default page size above the maximum is refused",
			env:         map[string]string{"ARTICLE_DEFAULT_PAGE_SIZE": "200"},
			wantProblem: "ARTICLE_DEFAULT_PAGE_SIZE",
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := article.LoadConfig(func(key string) (string, bool) { value, ok := tc.env[key]; return value, ok })

			if tc.wantProblem != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantProblem) {
					t.Fatalf("LoadConfig = %v, want an error naming %s", err, tc.wantProblem)
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
