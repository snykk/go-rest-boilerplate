package rest

import (
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// Routes registers the comment endpoints relative to the prefixes of the groups they go on. On public goes the
// list, GET /comments, which serves anyone. On signedIn go the endpoints that act for the caller, POST /comments
// and DELETE /comments/{id}; the group must put the caller's auth.Principal in the request's context, as
// auth.Authenticate does, or they answer 401.
func (h *Handler) Routes(public, signedIn *httpserver.Router) {
	public.HandleFunc(http.MethodGet, "/comments", h.list)

	signedIn.HandleFunc(http.MethodPost, "/comments", h.add)
	signedIn.HandleFunc(http.MethodDelete, "/comments/{id}", h.delete)
}
