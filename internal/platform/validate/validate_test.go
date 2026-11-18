package validate_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/validate"
)

type sample struct {
	Title string `json:"title" validate:"required"`
	Slug  string `json:"slug" validate:"max=5"`
}

// Struct reports one error per invalid field, under the field's JSON name, without the submitted value.
func TestStruct(t *testing.T) {
	tCases := []struct {
		name        string
		in          any
		want        validate.Errors // nil means valid
		mustNotEcho string          // submitted value the error must not contain
	}{
		{name: "a valid struct passes", in: sample{Title: "t", Slug: "ok"}},
		{
			name: "a field is reported under its JSON name and rule",
			in:   sample{Slug: "ok"},
			want: validate.Errors{{Field: "title", Rule: "required", Message: "is required"}},
		},
		{
			name: "the submitted value is never echoed",
			in:   sample{Title: "t", Slug: "super-secret-value"},
			want: validate.Errors{{
				Field:   "slug",
				Rule:    "max",
				Message: "must be at most 5 characters",
			}},
			mustNotEcho: "super-secret-value",
		},
		{
			name: "every invalid field is reported",
			in:   sample{Slug: "too-long"},
			want: validate.Errors{
				{Field: "title", Rule: "required", Message: "is required"},
				{Field: "slug", Rule: "max", Message: "must be at most 5 characters"},
			},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validate.Struct(tc.in)

			var got validate.Errors
			if err != nil && !errors.As(err, &got) {
				t.Fatalf("Struct = %v, want validate.Errors", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Struct = %+v, want %+v", got, tc.want)
			}
			if tc.mustNotEcho != "" && strings.Contains(err.Error(), tc.mustNotEcho) {
				t.Errorf("Struct error = %q, want it without %q", err, tc.mustNotEcho)
			}
		})
	}
}
