package rest

import (
	"net/http"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// Routes registers the article endpoints relative to the prefixes of the groups they go on. On read go the
// public reads, which serve published articles only: GET /articles and GET /articles/{id}. On write go the
// changes, POST /articles, PATCH /articles/{id}, POST /articles/{id}/publish and DELETE /articles/{id}, and
// the reads of drafts, GET /articles/drafts and GET /articles/drafts/{id}. A service that lets only some callers
// change articles puts its authentication on write; read and write may be the same group.
func (h *Handler) Routes(read, write *httpserver.Router) {
	read.HandleFunc(http.MethodGet, "/articles", h.list)
	read.HandleFunc(http.MethodGet, "/articles/{id}", h.get)

	write.HandleFunc(http.MethodPost, "/articles", h.create)
	write.HandleFunc(http.MethodPatch, "/articles/{id}", h.update)
	write.HandleFunc(http.MethodPost, "/articles/{id}/publish", h.publish)
	write.HandleFunc(http.MethodDelete, "/articles/{id}", h.delete)
	write.HandleFunc(http.MethodGet, "/articles/drafts", h.listDrafts)
	write.HandleFunc(http.MethodGet, "/articles/drafts/{id}", h.getDraft)
}
