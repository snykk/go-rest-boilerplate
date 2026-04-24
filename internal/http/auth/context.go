// Package auth adapts the JWT claim carried in gin.Context to the
// opaque CurrentUser value handlers need. Handlers were previously
// type-asserting jwt.JwtCustomClaim directly, which coupled the HTTP
// layer to the JWT implementation and crashed (instead of returning
// 401) when the middleware mis-populated the context key.
package auth

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
)

// CurrentUser is the HTTP-layer view of an authenticated request.
type CurrentUser struct {
	ID      string
	Email   string
	IsAdmin bool
	JTI     string
}

// ErrNotAuthenticated means the auth middleware did not populate the
// expected context key (either auth middleware wasn't installed on the
// route, or the token was rejected upstream).
var ErrNotAuthenticated = errors.New("request is not authenticated")

// CurrentUserFromContext extracts the authenticated user from the
// gin.Context. Returns ErrNotAuthenticated when the context has no
// recognizable claims; handlers should respond with 401 in that case.
func CurrentUserFromContext(c *gin.Context) (CurrentUser, error) {
	raw, ok := c.Get(constants.CtxAuthenticatedUserKey)
	if !ok {
		return CurrentUser{}, ErrNotAuthenticated
	}
	claims, ok := raw.(jwt.JwtCustomClaim)
	if !ok {
		return CurrentUser{}, ErrNotAuthenticated
	}
	return CurrentUser{
		ID:      claims.UserID,
		Email:   claims.Email,
		IsAdmin: claims.IsAdmin,
		JTI:     claims.ID,
	}, nil
}
