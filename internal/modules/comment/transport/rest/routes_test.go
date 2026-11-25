package rest_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/transport/rest"
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

// Routes mounts the list, which serves anyone, on the public group, and adding and deleting a comment, which
// act for the caller, on the signed-in group.
func TestHandlerRoutes(t *testing.T) {
	tCases := []struct {
		name   string
		method string
		target string
		want   string
	}{
		{name: "listing is public", method: http.MethodGet, target: "/v1/comments", want: "public"},
		{name: "adding needs a caller", method: http.MethodPost, target: "/v1/comments", want: "signed-in"},
		{
			name:   "deleting needs a caller",
			method: http.MethodDelete,
			target: "/v1/comments/" + commentID,
			want:   "signed-in",
		},
	}

	router := httpserver.NewRouter()
	comments := rest.NewHandler(NewMockUseCases(gomock.NewController(t)), slog.New(slog.NewTextHandler(io.Discard, nil)))
	comments.Routes(router.Group("/v1", answerAs("public")), router.Group("/v1", answerAs("signed-in")))
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
