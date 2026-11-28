package config_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
	"github.com/snykk/go-rest-boilerplate/internal/platform/logger"
)

// NewSecret keeps the value for Reveal, and gives the zero Secret for an empty value.
func TestNewSecret(t *testing.T) {
	tCases := []struct {
		name     string
		secret   config.Secret
		want     string
		wantZero bool
	}{
		{name: "a value is revealed as given", secret: config.NewSecret("hunter2"), want: "hunter2"},
		{name: "the zero secret reveals the empty string", secret: config.Secret{}, want: "", wantZero: true},
		{name: "an empty value gives the zero secret", secret: config.NewSecret(""), want: "", wantZero: true},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.secret.Reveal(); got != tc.want {
				t.Errorf("Reveal() = %q, want %q", got, tc.want)
			}
			if zero := reflect.DeepEqual(tc.secret, config.Secret{}); zero != tc.wantZero {
				t.Errorf("reflect.DeepEqual(secret, config.Secret{}) = %t, want %t", zero, tc.wantZero)
			}
		})
	}
}

// Secret shows [REDACTED] to fmt and encoding/json, whatever the verb, and nothing for an empty secret.
func TestSecret(t *testing.T) {
	secret := config.NewSecret("hunter2")
	holder := struct{ Token config.Secret }{secret}

	tCases := []struct {
		name   string
		render func() string
		want   string
	}{
		{name: "the v verb is redacted", render: func() string { return fmt.Sprintf("%v", secret) }, want: logger.Redacted},
		{
			name:   "the plus v verb is redacted",
			render: func() string { return fmt.Sprintf("%+v", secret) },
			want:   logger.Redacted,
		},
		{
			name:   "the hash v verb is redacted",
			render: func() string { return fmt.Sprintf("%#v", secret) },
			want:   logger.Redacted,
		},
		{name: "the s verb is redacted", render: func() string { return fmt.Sprintf("%s", secret) }, want: logger.Redacted},
		{name: "the q verb is redacted", render: func() string { return fmt.Sprintf("%q", secret) }, want: logger.Redacted},
		{name: "the x verb is redacted", render: func() string { return fmt.Sprintf("%x", secret) }, want: logger.Redacted},
		{name: "the d verb is redacted", render: func() string { return fmt.Sprintf("%d", secret) }, want: logger.Redacted},
		{
			name:   "a struct holding a secret prints it redacted",
			render: func() string { return fmt.Sprintf("%+v", holder) },
			want:   "{Token:" + logger.Redacted + "}",
		},
		{
			name: "json encodes a secret redacted",
			render: func() string {
				encoded, err := json.Marshal(holder)
				if err != nil {
					t.Fatalf("json.Marshal = %v, want no error", err)
				}
				return string(encoded)
			},
			want: `{"Token":"` + logger.Redacted + `"}`,
		},
		{
			name:   "an empty secret prints nothing",
			render: func() string { return fmt.Sprintf("%v", config.Secret{}) },
			want:   "",
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.render(); got != tc.want {
				t.Errorf("rendered = %q, want %q", got, tc.want)
			}
		})
	}
}

// A Secret held in an unexported field stays hidden from fmt and slog, which reach such a field by reflection.
func TestSecretInUnexportedField(t *testing.T) {
	type holder struct{ token config.Secret }
	secret := config.NewSecret("hunter2")

	logged := func(handler func(*bytes.Buffer) slog.Handler) func() string {
		return func() string {
			var out bytes.Buffer
			slog.New(handler(&out)).Info("start", "token", secret, "holder", holder{secret})
			return out.String()
		}
	}

	tCases := []struct {
		name   string
		render func() string
	}{
		{name: "plus v of a struct value", render: func() string { return fmt.Sprintf("%+v", holder{secret}) }},
		{name: "hash v of a struct value", render: func() string { return fmt.Sprintf("%#v", holder{secret}) }},
		{name: "plus v of a struct pointer", render: func() string { return fmt.Sprintf("%+v", &holder{secret}) }},
		{name: "hash v of a struct pointer", render: func() string { return fmt.Sprintf("%#v", &holder{secret}) }},
		{
			name:   "a slog text record",
			render: logged(func(out *bytes.Buffer) slog.Handler { return slog.NewTextHandler(out, nil) }),
		},
		{
			name:   "a slog json record",
			render: logged(func(out *bytes.Buffer) slog.Handler { return slog.NewJSONHandler(out, nil) }),
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.render(); strings.Contains(got, "hunter2") {
				t.Errorf("rendered = %q, want no secret value in it", got)
			}
		})
	}
}
