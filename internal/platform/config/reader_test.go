package config_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
)

// Reader returns each setting or its default, and reports every invalid or missing setting at once, naming
// its key.
func TestReader(t *testing.T) {
	set := map[string]string{
		"NAME":         " x ",
		"TIMEOUT":      "2s",
		"BAD_DURATION": "abc",
		"SIZE":         "5",
		"ZERO":         "0",
		"FLAG":         "true",
		"MODE":         "c",
		"LIST":         " a, ,b ",
		"RATIO":        "0.25",
		"BAD_RATIO":    "1.5",
		"ENDPOINT":     "http://collector:4318",
		"BAD_URL":      "collector:4318",
	}
	source := config.Source(func(key string) (string, bool) { value, ok := set[key]; return value, ok })
	tCases := []struct {
		name        string
		read        func(reader *config.Reader) any
		want        any
		wantProblem string // the key the error must name; "" means no error
	}{
		{
			name: "an unset setting takes its default",
			read: func(reader *config.Reader) any { return reader.String("UNSET", "d") },
			want: "d",
		},
		{
			name: "a set setting is returned without surrounding spaces",
			read: func(reader *config.Reader) any { return reader.String("NAME", "d") },
			want: "x",
		},
		{
			name:        "a missing required setting is reported",
			read:        func(reader *config.Reader) any { return reader.Required("UNSET") },
			want:        "",
			wantProblem: "UNSET",
		},
		{
			name: "a duration is parsed",
			read: func(reader *config.Reader) any { return reader.Duration("TIMEOUT", time.Second) },
			want: 2 * time.Second,
		},
		{
			name:        "a malformed duration is reported and takes its default",
			read:        func(reader *config.Reader) any { return reader.Duration("BAD_DURATION", time.Second) },
			want:        time.Second,
			wantProblem: "BAD_DURATION",
		},
		{
			name: "a positive integer is parsed",
			read: func(reader *config.Reader) any { return reader.PositiveInt("SIZE", 1) },
			want: int64(5),
		},
		{
			name:        "zero is reported as not positive and takes the default",
			read:        func(reader *config.Reader) any { return reader.PositiveInt("ZERO", 1) },
			want:        int64(1),
			wantProblem: "ZERO",
		},
		{
			name: "a boolean is parsed",
			read: func(reader *config.Reader) any { return reader.Bool("FLAG", false) },
			want: true,
		},
		{
			name:        "a value outside the allowed ones is reported and takes the default",
			read:        func(reader *config.Reader) any { return reader.OneOf("MODE", "a", "a", "b") },
			want:        "a",
			wantProblem: "MODE",
		},
		{
			name: "a list drops blank entries and spaces",
			read: func(reader *config.Reader) any { return reader.List("LIST") },
			want: []string{"a", "b"},
		},
		{
			name: "a ratio from 0 to 1 is parsed",
			read: func(reader *config.Reader) any { return reader.Ratio("RATIO", 1) },
			want: 0.25,
		},
		{
			name:        "a ratio outside 0 to 1 is reported and takes its default",
			read:        func(reader *config.Reader) any { return reader.Ratio("BAD_RATIO", 1) },
			want:        1.0,
			wantProblem: "BAD_RATIO",
		},
		{
			name: "a URL with a scheme and a host is returned",
			read: func(reader *config.Reader) any { return reader.URL("ENDPOINT") },
			want: "http://collector:4318",
		},
		{
			name: "an unset URL is empty",
			read: func(reader *config.Reader) any { return reader.URL("UNSET") },
			want: "",
		},
		{
			name:        "a URL without an http or https scheme is reported",
			read:        func(reader *config.Reader) any { return reader.URL("BAD_URL") },
			want:        "",
			wantProblem: "BAD_URL",
		},
		{
			name:        "a rule the caller checks is reported",
			read:        func(reader *config.Reader) any { reader.Fail("CUSTOM", "must not exceed MAX"); return nil },
			wantProblem: "CUSTOM",
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := config.NewReader(source)

			got := tc.read(reader)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("read = %#v, want %#v", got, tc.want)
			}
			err := reader.Err()
			switch {
			case tc.wantProblem == "" && err != nil:
				t.Errorf("Err = %v, want nil", err)
			case tc.wantProblem != "" && (err == nil || !strings.Contains(err.Error(), tc.wantProblem)):
				t.Errorf("Err = %v, want it to name %s", err, tc.wantProblem)
			}
		})
	}
}
