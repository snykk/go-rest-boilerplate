package rest_test

import (
	"errors"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// Create answers 201 with the stored article and its location, under the prefix its routes are mounted on. It
// refuses a malformed or invalid body before the use cases are called, and maps their errors to problems; an
// unexpected error's cause is logged and never returned.
func TestHandlerCreate(t *testing.T) {
	const created = `{"id":"` + articleID + `","slug":"hello","title":"Hello","body":"","status":"draft",
		"published_at":null,"created_at":"2026-11-18T09:00:00Z","updated_at":"2026-11-18T09:00:00Z"}`
	expectCreated := func(f *fixture) {
		f.useCases.EXPECT().Create(gomock.Any(), usecase.CreateInput{
			Slug:  "hello",
			Title: "Hello",
		}).Return(viewFixture(), nil)
	}
	tCases := []struct {
		name         string
		target       string // "" means /v1/articles
		body         string
		mock         func(f *fixture)
		wantStatus   int
		wantCode     string // problem code; "" on success
		wantLocation string
		wantBody     string // the JSON body on success
		wantLogged   string // text the log must hold; "" means none is checked
	}{
		{
			name:         "a valid body creates the article",
			body:         `{"slug":"hello","title":"Hello"}`,
			mock:         expectCreated,
			wantStatus:   201,
			wantLocation: "/v1/articles/drafts/" + articleID,
			wantBody:     created,
		},
		{
			name:         "the location is under the prefix the routes are mounted on",
			target:       "/v2/articles",
			body:         `{"slug":"hello","title":"Hello"}`,
			mock:         expectCreated,
			wantStatus:   201,
			wantLocation: "/v2/articles/drafts/" + articleID,
			wantBody:     created,
		},
		{
			name:       "a missing title is refused before the service is called",
			body:       `{"slug":"hello"}`,
			mock:       func(*fixture) {},
			wantStatus: 422,
			wantCode:   "validation_failed",
		},
		{
			name:       "an unknown field is refused before the service is called",
			body:       `{"slug":"hello","title":"t","x":1}`,
			mock:       func(*fixture) {},
			wantStatus: 400,
			wantCode:   "malformed_json",
		},
		{
			name:       "a used slug is a conflict",
			body:       `{"slug":"hello","title":"t"}`,
			mock:       expectCreateFails(domain.ErrSlugTaken),
			wantStatus: 409,
			wantCode:   "slug_taken",
		},
		{
			name:       "an invalid slug is unprocessable",
			body:       `{"slug":"Hello","title":"t"}`,
			mock:       expectCreateFails(domain.ErrInvalidSlug),
			wantStatus: 422,
			wantCode:   "invalid_slug",
		},
		{
			name:       "a body with a NUL character is unprocessable",
			body:       `{"slug":"hello","title":"t","body":"a\u0000b"}`,
			mock:       expectCreateFails(domain.ErrInvalidBody),
			wantStatus: 422,
			wantCode:   "invalid_body",
		},
		{
			name:       "an unexpected error is an internal error that does not leak its cause",
			body:       `{"slug":"hello","title":"t"}`,
			mock:       expectCreateFails(errors.New("dial tcp 10.0.0.1:5432: refused")),
			wantStatus: 500,
			wantCode:   "internal",
			wantLogged: "dial tcp 10.0.0.1:5432: refused",
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)
			target := tc.target
			if target == "" {
				target = "/v1/articles"
			}

			recorder := f.serve("POST", target, tc.body)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", recorder.Code, tc.wantStatus, recorder.Body)
			}
			if strings.Contains(recorder.Body.String(), "10.0.0.1") {
				t.Errorf("body = %s, want the cause kept out", recorder.Body)
			}
			if tc.wantLogged != "" && !strings.Contains(f.logs.String(), tc.wantLogged) {
				t.Errorf("logs = %s, want the cause %q logged", f.logs, tc.wantLogged)
			}
			if tc.wantCode != "" {
				if got := problemCode(t, recorder); got != tc.wantCode {
					t.Errorf("problem code = %q, want %q", got, tc.wantCode)
				}
				return
			}
			if location := recorder.Header().Get("Location"); location != tc.wantLocation {
				t.Errorf("Location = %q, want %q", location, tc.wantLocation)
			}
			if !sameJSON(t, recorder.Body.String(), tc.wantBody) {
				t.Errorf("body = %s, want %s", recorder.Body, tc.wantBody)
			}
		})
	}
}
