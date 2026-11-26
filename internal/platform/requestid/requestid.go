// Package requestid carries the ID of the current request in a context.
package requestid

import "context"

// Header is the HTTP header that carries a request ID.
const Header = "X-Request-ID"

type ctxKey struct{}

// With returns a copy of ctx that carries id.
func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// From returns the request ID stored in ctx, or "" when there is none.
func From(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// Valid reports whether id is 1 to 64 characters of A–Z, a–z, 0–9, '.', '_' and '-'.
func Valid(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for i := range len(id) {
		switch character := id[i]; {
		case 'a' <= character && character <= 'z', 'A' <= character && character <= 'Z',
			'0' <= character && character <= '9', character == '.', character == '_', character == '-':
		default:
			return false
		}
	}
	return true
}
