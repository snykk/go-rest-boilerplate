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

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/transport/rest"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

var baseTime = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)

const (
	commentID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c44"
	articleID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c45"
	readerID  = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c46"
	editorID  = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c47"
	// submittedBody is the text of the comment the requests send; no problem or log record may repeat it.
	submittedBody = "Thanks, this helped."
)

// commentJSON is viewFixture as the API returns it.
const commentJSON = `{"id":"` + commentID + `","article_id":"` + articleID + `","author_id":"` + readerID + `",
	"author_name":"Ada","body":"Thanks, this helped.","created_at":"2026-11-25T09:00:00Z"}`

// errStoreDown is a store failing on its own; its cause, the address, must never reach a response.
var errStoreDown = errors.New("dial tcp 10.0.0.1:5432: connection refused")

// viewFixture is the comment the reader wrote on the article at baseTime, as the use cases return it.
func viewFixture() usecase.CommentView {
	return usecase.CommentView{
		ID:         commentID,
		ArticleID:  articleID,
		AuthorID:   readerID,
		AuthorName: "Ada",
		Body:       submittedBody,
		CreatedAt:  baseTime,
	}
}

// reader is a signed-in reader, the author of viewFixture.
func reader() *auth.Principal {
	return &auth.Principal{ID: readerID, Roles: []string{"reader"}}
}

// editor is a signed-in editor.
func editor() *auth.Principal {
	return &auth.Principal{ID: editorID, Roles: []string{"reader", "editor"}}
}

// fixture is a Handler over a mocked UseCases, with its public and signed-in endpoints both mounted on the /v1
// group of a router of its own. Who calls is up to each request, as auth.Authenticate would set it. The handler
// logs into logs.
type fixture struct {
	useCases *MockUseCases
	logs     *bytes.Buffer
	handler  http.Handler
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{useCases: NewMockUseCases(gomock.NewController(t)), logs: &bytes.Buffer{}}
	router := httpserver.NewRouter()
	comments := rest.NewHandler(f.useCases, slog.New(slog.NewJSONHandler(f.logs, nil)))
	v1 := router.Group("/v1")
	comments.Routes(v1, v1)
	f.handler = router.Handler()
	return f
}

// serve sends one request through the router, from caller unless caller is nil, and records the response.
func (f *fixture) serve(caller *auth.Principal, method, target, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if caller != nil {
		request = request.WithContext(auth.WithPrincipal(request.Context(), *caller))
	}
	recorder := httptest.NewRecorder()
	f.handler.ServeHTTP(recorder, request)
	return recorder
}

// outcome is what a case expects of a response.
type outcome struct {
	status int
	code   string // the problem code; "" on success
	field  string // the one field a validation_failed problem names; "" when none is checked
	body   string // the JSON body on success; "" means no body
	logged string // text the log must hold; "" means none is checked
}

// check compares the response recorder holds with want. It also fails when a problem or the log repeats the
// submitted comment, or when the response carries the cause of an unexpected error.
func (f *fixture) check(t *testing.T, recorder *httptest.ResponseRecorder, want outcome) {
	t.Helper()
	if recorder.Code != want.status {
		t.Fatalf("status = %d, want %d; body %s", recorder.Code, want.status, recorder.Body)
	}
	if strings.Contains(recorder.Body.String(), "10.0.0.1") {
		t.Errorf("body = %s, want the cause kept out", recorder.Body)
	}
	if strings.Contains(f.logs.String(), submittedBody) {
		t.Errorf("logs = %s, want the submitted comment kept out", f.logs)
	}
	if want.logged != "" && !strings.Contains(f.logs.String(), want.logged) {
		t.Errorf("logs = %s, want %q logged", f.logs, want.logged)
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

// checkProblem compares the problem recorder carries with the code and the field want names, and fails when it
// repeats the submitted comment.
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
	if strings.Contains(recorder.Body.String(), submittedBody) {
		t.Errorf("body = %s, want the submitted comment kept out", recorder.Body)
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
