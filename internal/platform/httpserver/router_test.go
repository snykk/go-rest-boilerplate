package httpserver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// Handle registers a route under its group's prefix and wraps it in the middleware of every group it was
// registered through, outer groups first. A pattern registered twice panics.
func TestRouterHandle(t *testing.T) {
	var trail []string
	mark := func(name string) httpserver.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				trail = append(trail, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	reach := func(name string) http.HandlerFunc {
		return func(http.ResponseWriter, *http.Request) { trail = append(trail, name) }
	}
	router := httpserver.NewRouter()
	router.HandleFunc("GET", "/health", reach("health"))
	v1 := router.Group("/v1", mark("v1"))
	v1.HandleFunc("GET", "/things/{id}", reach("thing"))
	v1.Group("/admin", mark("admin")).HandleFunc("DELETE", "/things/{id}", reach("delete"))
	handler := router.Handler()
	tCases := []struct {
		name, method, path string
		wantStatus         int
		wantTrail          []string
	}{
		{
			name:       "a root route runs without group middleware",
			method:     "GET",
			path:       "/health",
			wantStatus: 200,
			wantTrail:  []string{"health"},
		},
		{
			name:       "a group route lives under the prefix and runs the group middleware",
			method:     "GET",
			path:       "/v1/things/1",
			wantStatus: 200,
			wantTrail:  []string{"v1", "thing"},
		},
		{
			name:       "a nested group adds its prefix and middleware after the outer group's",
			method:     "DELETE",
			path:       "/v1/admin/things/1",
			wantStatus: 200,
			wantTrail:  []string{"v1", "admin", "delete"},
		},
		{name: "a group prefix alone is not a route", method: "GET", path: "/v1", wantStatus: 404},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			trail = nil
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, nil))

			if recorder.Code != tc.wantStatus || !reflect.DeepEqual(trail, tc.wantTrail) {
				t.Errorf("%s %s = %d via %v, want %d via %v",
					tc.method, tc.path, recorder.Code, trail, tc.wantStatus, tc.wantTrail)
			}
		})
	}
	t.Run("a pattern registered twice panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("Handle accepted a pattern that is already registered")
			}
		}()
		v1.HandleFunc("GET", "/things/{id}", reach("again"))
	})
}

// Handler serves the registered routes and answers a path or method no route takes with a problem; the
// mux's redirects pass through.
func TestRouterHandler(t *testing.T) {
	router := httpserver.NewRouter()
	router.HandleFunc("GET", "/v1/things/{id}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	handler := router.Handler()
	tCases := []struct {
		name         string
		method, path string
		wantStatus   int
		wantCode     string // problem code; "" means the response is not a problem
		wantAllow    string
	}{
		{name: "a matched route is served", method: "GET", path: "/v1/things/1", wantStatus: 204},
		{
			name:       "a path no route matches is a not_found problem",
			method:     "GET",
			path:       "/v1/nope",
			wantStatus: 404,
			wantCode:   "not_found",
		},
		{
			name:       "a method the route does not take is a method_not_allowed problem with the allowed methods",
			method:     "PUT",
			path:       "/v1/things/1",
			wantStatus: 405,
			wantCode:   "method_not_allowed",
			wantAllow:  "GET, HEAD",
		},
		{
			name:       "a path that needs cleaning is still redirected",
			method:     "GET",
			path:       "/v1/nope//x",
			wantStatus: http.StatusTemporaryRedirect,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, nil))

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", recorder.Code, tc.wantStatus, recorder.Body)
			}
			if allow := recorder.Header().Get("Allow"); allow != tc.wantAllow {
				t.Errorf("Allow = %q, want %q", allow, tc.wantAllow)
			}
			if tc.wantCode == "" {
				return
			}
			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/problem+json" {
				t.Errorf("Content-Type = %q, want application/problem+json", contentType)
			}
			var problem httpserver.Problem
			if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil || problem.Code != tc.wantCode {
				t.Errorf("body = %s (%v), want a %s problem", recorder.Body, err, tc.wantCode)
			}
		})
	}
}
