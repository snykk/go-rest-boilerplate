// Package auth serves the /auth/* HTTP endpoints — register, login,
// OTP, refresh, logout. User-profile endpoints live in the sibling
// package internal/http/handlers/v1/users.
package auth

import (
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
)

// Handler is the auth-handler aggregate; per-endpoint methods are
// defined in their own files (auth.register.go, auth.login.go, etc.)
// so PR diffs touching one endpoint don't bleed into others.
type Handler struct {
	usecase auth.Usecase
}

func NewHandler(usecase auth.Usecase) Handler {
	return Handler{usecase: usecase}
}
