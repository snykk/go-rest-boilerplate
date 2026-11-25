package rest_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/transport/rest"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

var baseTime = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)

const (
	userID    = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c44"
	userEmail = "ada@example.com"
	// secretPassword is the password the requests send; no response or log record may repeat it.
	secretPassword = "Secret-Value-1"
	// newSecretPassword is the new password of a change; no response or log record may repeat it.
	newSecretPassword = "Secret-Value-2"
	verifyCode        = "123456"
	resetToken        = "reset-token-1"
	refreshToken      = "refresh-token-1"
	accessToken       = "access-token-1"
)

// userJSON is viewFixture as the API returns it.
const userJSON = `{"id":"` + userID + `","email":"ada@example.com","display_name":"Ada","roles":["reader"],
	"verified":false,"created_at":"2026-11-25T09:00:00Z"}`

// pairJSON is pairFixture as the API returns it.
const pairJSON = `{"access_token":"access-token-1","access_expires_at":"2026-11-25T09:15:00Z",
	"refresh_token":"refresh-token-1","refresh_expires_at":"2026-12-25T09:00:00Z","token_type":"Bearer"}`

// errStoreDown is a store failing on its own; its cause, the address, must never reach a response.
var errStoreDown = errors.New("dial tcp 10.0.0.1:5432: connection refused")

// viewFixture is an unverified reader named Ada, created at baseTime, as the use cases return them.
func viewFixture() usecase.UserView {
	return usecase.UserView{
		ID:          userID,
		Email:       userEmail,
		DisplayName: "Ada",
		Roles:       []string{"reader"},
		CreatedAt:   baseTime,
	}
}

// pairFixture is a token pair as the use cases return it: an access token for 15 minutes and a refresh token
// for 30 days.
func pairFixture() usecase.TokenPair {
	return usecase.TokenPair{
		AccessToken:      accessToken,
		AccessExpiresAt:  baseTime.Add(15 * time.Minute),
		RefreshToken:     refreshToken,
		RefreshExpiresAt: baseTime.Add(720 * time.Hour),
	}
}

// fixture is a Handler over a mocked UseCases, mounted on a router of its own as the application mounts it: the
// public endpoints on /v1, and the signed-in ones on a /v1 group whose requests come from userID, as
// auth.Authenticate passes them on. On /v2 every endpoint is mounted with no caller, as on a group without
// authentication. The handler logs into logs.
type fixture struct {
	useCases *MockUseCases
	logs     *bytes.Buffer
	handler  http.Handler
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{useCases: NewMockUseCases(gomock.NewController(t)), logs: &bytes.Buffer{}}
	router := httpserver.NewRouter()
	users := rest.NewHandler(f.useCases, slog.New(slog.NewJSONHandler(f.logs, nil)))
	v1, v2 := router.Group("/v1"), router.Group("/v2")
	users.Routes(v1, v1.Group("", signedInAs(userID)))
	users.Routes(v2, v2)
	f.handler = router.Handler()
	return f
}

// signedInAs is middleware that passes every request on with a reader whose ID is id as its principal.
func signedInAs(id string) httpserver.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal := auth.Principal{ID: id, Roles: []string{"reader"}}
			next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), principal)))
		})
	}
}

// serve sends one request through the router and records the response.
func (f *fixture) serve(method, target, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	f.handler.ServeHTTP(recorder, httptest.NewRequest(method, target, strings.NewReader(body)))
	return recorder
}

// outcome is what a case expects of a response.
type outcome struct {
	status int
	code   string // the problem code; "" on success
	field  string // the one field a validation_failed problem names; "" when none is checked
	body   string // the JSON body on success; "" means no body
	// noStore requires "Cache-Control: no-store", which keeps tokens out of caches.
	noStore bool
	logged  string // text the log must hold; "" means none is checked
}

// check compares the response recorder holds with want. It also fails when the response or the log repeats a
// password the requests send, or the cause of an unexpected error.
func (f *fixture) check(t *testing.T, recorder *httptest.ResponseRecorder, want outcome) {
	t.Helper()
	if recorder.Code != want.status {
		t.Fatalf("status = %d, want %d; body %s", recorder.Code, want.status, recorder.Body)
	}
	for _, secret := range []string{secretPassword, newSecretPassword, "10.0.0.1"} {
		if strings.Contains(recorder.Body.String(), secret) {
			t.Errorf("body = %s, want %q kept out", recorder.Body, secret)
		}
	}
	for _, secret := range []string{secretPassword, newSecretPassword} {
		if strings.Contains(f.logs.String(), secret) {
			t.Errorf("logs = %s, want %q kept out", f.logs, secret)
		}
	}
	if want.logged != "" && !strings.Contains(f.logs.String(), want.logged) {
		t.Errorf("logs = %s, want %q logged", f.logs, want.logged)
	}
	if got := recorder.Header().Get("Cache-Control"); want.noStore && got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}

	if want.code != "" {
		checkProblem(t, recorder, want)
		return
	}
	if want.body == "" {
		if recorder.Body.Len() != 0 {
			t.Errorf("body = %s, want none", recorder.Body)
		}
		return
	}
	if !sameJSON(t, recorder.Body.String(), want.body) {
		t.Errorf("body = %s, want %s", recorder.Body, want.body)
	}
}

// checkProblem compares the problem recorder carries with the code and the field want names.
func checkProblem(t *testing.T, recorder *httptest.ResponseRecorder, want outcome) {
	t.Helper()
	var problem httpserver.Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("body %s is not a problem: %v", recorder.Body, err)
	}
	if problem.Code != want.code {
		t.Errorf("problem code = %q, want %q", problem.Code, want.code)
	}
	if want.field != "" && (len(problem.Errors) != 1 || problem.Errors[0].Field != want.field) {
		t.Errorf("problem errors = %+v, want one naming %q", problem.Errors, want.field)
	}
}

// sameJSON reports whether two JSON documents hold the same values.
func sameJSON(t *testing.T, got, want string) bool {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal([]byte(got), &gotValue); err != nil {
		t.Fatalf("got %s is not JSON: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("want %s is not JSON: %v", want, err)
	}
	return reflect.DeepEqual(gotValue, wantValue)
}
