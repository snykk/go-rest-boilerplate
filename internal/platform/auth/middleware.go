package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// ErrUnauthenticated means a request carries no credentials, or credentials the Authenticator refuses.
var ErrUnauthenticated = errors.New("unauthenticated")

// Authenticator identifies the caller of a request from its credentials.
type Authenticator interface {
	// Authenticate returns the caller of r. It returns an error wrapping ErrUnauthenticated when r carries no
	// credentials or credentials it refuses, and any other error when it cannot decide, such as when the
	// identity provider is down. It must be safe for concurrent use.
	Authenticate(r *http.Request) (Principal, error)
}

// AuthenticatorFunc adapts a function to an Authenticator.
type AuthenticatorFunc func(r *http.Request) (Principal, error)

// Authenticate calls f(r).
func (f AuthenticatorFunc) Authenticate(r *http.Request) (Principal, error) { return f(r) }

// Authenticate returns middleware that identifies the caller of each request with authenticator and passes the request
// on with the principal in its context (see PrincipalFrom). A request that authenticator refuses answers 401
// unauthenticated with the header "WWW-Authenticate: <challenge>", such as "Bearer". When authenticator cannot decide,
// the request answers 500 internal and the error is logged, never returned.
func Authenticate(authenticator Authenticator, challenge string, log *slog.Logger) httpserver.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := authenticator.Authenticate(r)
			switch {
			case errors.Is(err, ErrUnauthenticated):
				w.Header().Set("WWW-Authenticate", challenge)
				httpserver.WriteProblem(w, r, httpserver.Problem{
					Status: http.StatusUnauthorized,
					Code:   "unauthenticated",
					Detail: "the request needs valid credentials",
				})
			case err != nil:
				log.LogAttrs(r.Context(), slog.LevelError, "authentication failed", slog.String("error", err.Error()))
				httpserver.WriteProblem(w, r, httpserver.Problem{
					Status: http.StatusInternalServerError,
					Code:   "internal",
					Detail: "internal server error",
				})
			default:
				next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
			}
		})
	}
}

// Require returns middleware that passes a request on only when allow accepts its principal, and answers 403
// forbidden otherwise. A request without a principal is refused too, so Require is mounted after
// Authenticate.
func Require(allow func(Principal) bool) httpserver.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFrom(r.Context())
			if !ok || !allow(principal) {
				httpserver.WriteProblem(w, r, httpserver.Problem{
					Status: http.StatusForbidden,
					Code:   "forbidden",
					Detail: "the caller may not make this request",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Role returns a check for Require that accepts a principal with role.
func Role(role string) func(Principal) bool {
	return func(principal Principal) bool { return principal.HasRole(role) }
}
