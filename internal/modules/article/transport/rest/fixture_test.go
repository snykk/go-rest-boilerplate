package rest_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/transport/rest"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

var baseTime = time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)

const articleID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c44"

// viewFixture is a draft with slug "hello" and title "Hello", created at baseTime, as the use cases return it.
func viewFixture() usecase.ArticleView {
	return usecase.ArticleView{
		ID:        articleID,
		Slug:      "hello",
		Title:     "Hello",
		Status:    domain.StatusDraft,
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	}
}

// fixture is a Handler over a mocked UseCases, with its reads and changes mounted on the /v1 group of a router
// of its own, as the application mounts them, and on /v2, to show that the handler does not depend on its
// prefix. The handler logs into logs.
type fixture struct {
	useCases *MockUseCases
	logs     *bytes.Buffer
	handler  http.Handler
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{useCases: NewMockUseCases(gomock.NewController(t)), logs: &bytes.Buffer{}}
	router := httpserver.NewRouter()
	articles := rest.NewHandler(f.useCases, slog.New(slog.NewJSONHandler(f.logs, nil)))
	v1, v2 := router.Group("/v1"), router.Group("/v2")
	articles.Routes(v1, v1)
	articles.Routes(v2, v2)
	f.handler = router.Handler()
	return f
}

// serve sends one request through the router and records the response.
func (f *fixture) serve(method, target, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	f.handler.ServeHTTP(recorder, httptest.NewRequest(method, target, strings.NewReader(body)))
	return recorder
}

// problemCode is the code of the problem recorder carries.
func problemCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var problem httpserver.Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("body %s is not a problem: %v", recorder.Body, err)
	}
	return problem.Code
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

// expectCreateFails makes Create fail with err.
func expectCreateFails(err error) func(f *fixture) {
	return func(f *fixture) {
		f.useCases.EXPECT().Create(gomock.Any(), gomock.Any()).Return(usecase.ArticleView{}, err)
	}
}
