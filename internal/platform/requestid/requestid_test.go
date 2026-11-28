package requestid_test

import (
	"context"
	"strings"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/requestid"
)

// From returns the request ID With stored in a context, and "" when there is none.
func TestFrom(t *testing.T) {
	tCases := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{name: "a stored ID is returned", ctx: requestid.With(context.Background(), "rid-1"), want: "rid-1"},
		{name: "a context without one returns nothing", ctx: context.Background(), want: ""},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := requestid.From(tc.ctx); got != tc.want {
				t.Errorf("From = %q, want %q", got, tc.want)
			}
		})
	}
}

// Valid accepts 1 to 64 characters of letters, digits, dot, underscore and hyphen.
func TestValid(t *testing.T) {
	tCases := []struct {
		name string
		id   string
		want bool
	}{
		{name: "a single character is valid", id: "a", want: true},
		{name: "every allowed character is valid", id: "abc-123_X.9", want: true},
		{name: "64 characters are valid", id: strings.Repeat("a", 64), want: true},
		{name: "an empty ID is invalid", id: "", want: false},
		{name: "65 characters are invalid", id: strings.Repeat("a", 65), want: false},
		{name: "a space is invalid", id: "a b", want: false},
		{name: "markup is invalid", id: "<script>", want: false},
		{name: "a non-ASCII letter is invalid", id: "é", want: false},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := requestid.Valid(tc.id); got != tc.want {
				t.Errorf("Valid(%q) = %t, want %t", tc.id, got, tc.want)
			}
		})
	}
}
