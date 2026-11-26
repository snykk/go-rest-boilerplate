package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// StaticToken returns the authenticator of operator endpoints. It identifies as principal a request whose Authorization
// header is "Bearer <token>", the scheme in any case, and returns ErrUnauthenticated for every other request.
// It compares the token in constant time, and an empty token refuses every request. It is safe for concurrent
// use.
func StaticToken(token string, principal Principal) AuthenticatorFunc {
	want := []byte(token)
	return AuthenticatorFunc(func(r *http.Request) (Principal, error) {
		scheme, got, ok := strings.Cut(r.Header.Get("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || len(want) == 0 ||
			subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			return Principal{}, ErrUnauthenticated
		}
		return principal, nil
	})
}
