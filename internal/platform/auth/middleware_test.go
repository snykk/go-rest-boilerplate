package auth_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
)

var errKeyServerDown = errors.New("key server down")

// problemCode returns the code member of the problem in recorder's body.
func problemCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var problem struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("body %q is not a problem: %v", recorder.Body, err)
	}
	return problem.Code
}

// Authenticate passes the caller's principal on to the handler. It answers 401 with the challenge when the
// authenticator refuses the credentials, and 500 without the cause, which it logs, when the authenticator
// cannot decide.
func TestAuthenticate(t *testing.T) {
	alice := auth.Principal{ID: "alice"}
	tCases := []struct {
		name          string
		authenticate  auth.AuthenticatorFunc
		wantStatus    int
		wantCode      string // "" means the handler answered
		wantChallenge string
		wantLogged    bool
	}{
		{
			name:         "accepted credentials reach the handler with the principal",
			authenticate: func(*http.Request) (auth.Principal, error) { return alice, nil },
			wantStatus:   http.StatusOK,
		},
		{
			name:          "refused credentials answer 401 with the challenge",
			authenticate:  func(*http.Request) (auth.Principal, error) { return auth.Principal{}, auth.ErrUnauthenticated },
			wantStatus:    http.StatusUnauthorized,
			wantCode:      "unauthenticated",
			wantChallenge: "Bearer",
		},
		{
			name: "a refusal that wraps ErrUnauthenticated answers 401",
			authenticate: func(*http.Request) (auth.Principal, error) {
				return auth.Principal{}, fmt.Errorf("token expired: %w", auth.ErrUnauthenticated)
			},
			wantStatus:    http.StatusUnauthorized,
			wantCode:      "unauthenticated",
			wantChallenge: "Bearer",
		},
		{
			name:         "an authenticator that cannot decide answers 500 and logs the cause",
			authenticate: func(*http.Request) (auth.Principal, error) { return auth.Principal{}, errKeyServerDown },
			wantStatus:   http.StatusInternalServerError,
			wantCode:     "internal",
			wantLogged:   true,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			log := slog.New(slog.NewJSONHandler(&logs, nil))
			var got auth.Principal
			var reached bool
			handler := auth.Authenticate(tc.authenticate, "Bearer", log)(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) { got, reached = auth.PrincipalFrom(r.Context()) }))
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/articles", nil))

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.wantStatus)
			}
			if tc.wantCode == "" {
				if !reached || got.ID != alice.ID {
					t.Errorf("handler principal = %+v, %t; want %+v, true", got, reached, alice)
				}
				return
			}
			if reached {
				t.Error("handler ran, want the request stopped")
			}
			if code := problemCode(t, recorder); code != tc.wantCode {
				t.Errorf("problem code = %q, want %q", code, tc.wantCode)
			}
			if challenge := recorder.Header().Get("WWW-Authenticate"); challenge != tc.wantChallenge {
				t.Errorf("WWW-Authenticate = %q, want %q", challenge, tc.wantChallenge)
			}
			if logged := strings.Contains(logs.String(), errKeyServerDown.Error()); logged != tc.wantLogged {
				t.Errorf("cause logged = %t, want %t; logs: %s", logged, tc.wantLogged, logs.String())
			}
			if strings.Contains(recorder.Body.String(), errKeyServerDown.Error()) {
				t.Errorf("body = %s, want the cause kept out of the response", recorder.Body)
			}
		})
	}
}

// Require, with Role, lets a request through only when its principal has the role, and answers 403
// otherwise, also when the request carries no principal.
func TestRequire(t *testing.T) {
	tCases := []struct {
		name       string
		principal  *auth.Principal // nil means the request carries none
		wantStatus int
	}{
		{
			name:       "a principal with the role passes",
			principal:  &auth.Principal{ID: "alice", Roles: []string{"admin"}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "a principal without the role is forbidden",
			principal:  &auth.Principal{ID: "bob", Roles: []string{"editor"}},
			wantStatus: http.StatusForbidden,
		},
		{name: "a request without a principal is forbidden", wantStatus: http.StatusForbidden},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			var reached bool
			handler := auth.Require(auth.Role("admin"))(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) { reached = true }))
			request := httptest.NewRequest(http.MethodDelete, "/v1/articles/1", nil)
			if tc.principal != nil {
				request = request.WithContext(auth.WithPrincipal(request.Context(), *tc.principal))
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.wantStatus)
			}
			if reached != (tc.wantStatus == http.StatusOK) {
				t.Errorf("handler ran = %t, want %t", reached, tc.wantStatus == http.StatusOK)
			}
			if tc.wantStatus == http.StatusForbidden {
				if code := problemCode(t, recorder); code != "forbidden" {
					t.Errorf("problem code = %q, want %q", code, "forbidden")
				}
			}
		})
	}
}
