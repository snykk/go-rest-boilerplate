package rest

import (
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// Routes registers the identity endpoints relative to the prefixes of the groups they go on. On public go the
// endpoints that serve anyone: POST /auth/register, /auth/verification/resend, /auth/verification,
// /auth/login, /auth/refresh, /auth/logout, /auth/password/forgot and /auth/password/reset. On signedIn go the
// endpoints that act for the caller, POST /auth/password/change and GET /users/me; the group must put the
// caller's auth.Principal in the request's context, as auth.Authenticate does, or they answer 401.
func (h *Handler) Routes(public, signedIn *httpserver.Router) {
	public.HandleFunc(http.MethodPost, "/auth/register", h.register)
	public.HandleFunc(http.MethodPost, "/auth/verification/resend", h.resendVerification)
	public.HandleFunc(http.MethodPost, "/auth/verification", h.verify)
	public.HandleFunc(http.MethodPost, "/auth/login", h.login)
	public.HandleFunc(http.MethodPost, "/auth/refresh", h.refresh)
	public.HandleFunc(http.MethodPost, "/auth/logout", h.logout)
	public.HandleFunc(http.MethodPost, "/auth/password/forgot", h.forgotPassword)
	public.HandleFunc(http.MethodPost, "/auth/password/reset", h.resetPassword)

	signedIn.HandleFunc(http.MethodPost, "/auth/password/change", h.changePassword)
	signedIn.HandleFunc(http.MethodGet, "/users/me", h.me)
}
