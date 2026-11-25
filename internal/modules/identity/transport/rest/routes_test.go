package rest_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/transport/rest"
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

// Routes mounts the endpoints that serve anyone on the public group, and the two that act for the caller, the
// password change and the profile, on the signed-in group.
func TestHandlerRoutes(t *testing.T) {
	tCases := []struct {
		name   string
		method string
		target string
		want   string
	}{
		{name: "registering is public", method: http.MethodPost, target: "/v1/auth/register", want: "public"},
		{
			name:   "asking for a new code is public",
			method: http.MethodPost,
			target: "/v1/auth/verification/resend",
			want:   "public",
		},
		{name: "verifying is public", method: http.MethodPost, target: "/v1/auth/verification", want: "public"},
		{name: "signing in is public", method: http.MethodPost, target: "/v1/auth/login", want: "public"},
		{name: "refreshing is public", method: http.MethodPost, target: "/v1/auth/refresh", want: "public"},
		{name: "signing out is public", method: http.MethodPost, target: "/v1/auth/logout", want: "public"},
		{
			name:   "asking for a reset is public",
			method: http.MethodPost,
			target: "/v1/auth/password/forgot",
			want:   "public",
		},
		{name: "resetting is public", method: http.MethodPost, target: "/v1/auth/password/reset", want: "public"},
		{
			name:   "changing the password needs a caller",
			method: http.MethodPost,
			target: "/v1/auth/password/change",
			want:   "signed-in",
		},
		{name: "the profile needs a caller", method: http.MethodGet, target: "/v1/users/me", want: "signed-in"},
	}

	router := httpserver.NewRouter()
	users := rest.NewHandler(NewMockUseCases(gomock.NewController(t)), slog.New(slog.NewTextHandler(io.Discard, nil)))
	users.Routes(router.Group("/v1", answerAs("public")), router.Group("/v1", answerAs("signed-in")))
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
