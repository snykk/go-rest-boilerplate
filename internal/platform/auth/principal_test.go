package auth_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
)

// PrincipalFrom returns the principal a context carries, and reports false for a context without one.
func TestPrincipalFrom(t *testing.T) {
	alice := auth.Principal{ID: "alice", Roles: []string{"editor"}}
	tCases := []struct {
		name   string
		ctx    context.Context
		want   auth.Principal
		wantOK bool
	}{
		{
			name:   "a context with a principal gives it back",
			ctx:    auth.WithPrincipal(context.Background(), alice),
			want:   alice,
			wantOK: true,
		},
		{name: "a context without a principal reports none", ctx: context.Background()},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := auth.PrincipalFrom(tc.ctx)

			if ok != tc.wantOK || !reflect.DeepEqual(got, tc.want) {
				t.Errorf("PrincipalFrom = %+v, %t; want %+v, %t", got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

// HasRole reports whether the principal has the role, compared exactly.
func TestPrincipalHasRole(t *testing.T) {
	principal := auth.Principal{ID: "alice", Roles: []string{"editor", "admin"}}
	tCases := []struct {
		name string
		role string
		want bool
	}{
		{name: "a role the principal has", role: "admin", want: true},
		{name: "a role the principal lacks", role: "owner"},
		{name: "roles are compared exactly", role: "Admin"},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := principal.HasRole(tc.role); got != tc.want {
				t.Errorf("HasRole(%q) = %t, want %t", tc.role, got, tc.want)
			}
		})
	}
}
