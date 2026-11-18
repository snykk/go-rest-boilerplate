package rest_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/transport/rest"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// answerAs is a middleware that answers every request itself, naming group in the Group header.
func answerAs(group string) httpserver.Middleware {
	return func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Group", group)
			w.WriteHeader(http.StatusNoContent)
		})
	}
}

// Routes mounts the reads on the read group and the changes on the write group,
// so a service can put authentication on the changes alone.
func TestHandlerRoutes(t *testing.T) {
	tCases := []struct {
		name   string
		method string
		target string
		want   string
	}{
		{name: "listing is a read", method: http.MethodGet, target: "/v1/articles", want: "read"},
		{name: "getting one is a read", method: http.MethodGet, target: "/v1/articles/" + articleID, want: "read"},
		{name: "creating is a change", method: http.MethodPost, target: "/v1/articles", want: "write"},
		{name: "updating is a change", method: http.MethodPatch, target: "/v1/articles/" + articleID, want: "write"},
		{
			name:   "publishing is a change",
			method: http.MethodPost,
			target: "/v1/articles/" + articleID + "/publish",
			want:   "write",
		},
		{name: "deleting is a change", method: http.MethodDelete, target: "/v1/articles/" + articleID, want: "write"},
		{name: "listing drafts is a change", method: http.MethodGet, target: "/v1/articles/drafts", want: "write"},
		{
			name:   "getting a draft is a change",
			method: http.MethodGet,
			target: "/v1/articles/drafts/" + articleID,
			want:   "write",
		},
	}

	router := httpserver.NewRouter()
	articles := rest.NewHandler(NewMockUseCases(gomock.NewController(t)), slog.New(slog.NewTextHandler(io.Discard, nil)))
	articles.Routes(router.Group("/v1", answerAs("read")), router.Group("/v1", answerAs("write")))
	handler := router.Handler()

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.target, nil))

			if got := recorder.Header().Get("Group"); got != tc.want {
				t.Errorf("%s %s went to group %q (status %d), want %q", tc.method, tc.target, got, recorder.Code, tc.want)
			}
		})
	}
}
