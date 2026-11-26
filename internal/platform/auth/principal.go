// Package auth carries the authenticated caller of a request and guards routes by it. It does not check
// credentials itself: an Authenticator, such as a JWT or session verifier, does that.
package auth

import (
	"context"
	"slices"
)

// Principal is the authenticated caller of a request: a user, or another service.
type Principal struct {
	// ID identifies the caller, such as a user ID.
	ID string
	// Roles are the caller's roles, such as "admin".
	Roles []string
}

// HasRole reports whether p has role, compared exactly.
func (p Principal) HasRole(role string) bool {
	return slices.Contains(p.Roles, role)
}

type principalKey struct{}

// WithPrincipal returns a copy of ctx that carries principal.
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

// PrincipalFrom returns the principal ctx carries, and false when it carries none.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok
}
