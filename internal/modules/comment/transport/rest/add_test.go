package rest_test

import (
	"net/http"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
)

// Add answers 201 with the comment the signed-in caller wrote, and 401 to a request with no caller. It refuses a
// malformed body, and an article_id that is missing or not a UUID, before the use cases are called, and maps
// their errors to problems that never repeat the comment; an unexpected error's cause is logged and never
// returned.
func TestHandlerAdd(t *testing.T) {
	const body = `{"article_id":"` + articleID + `","body":"` + submittedBody + `"}`
	expectAdded := func(input usecase.AddInput) func(f *fixture) {
		return func(f *fixture) { f.useCases.EXPECT().Add(gomock.Any(), input).Return(viewFixture(), nil) }
	}
	expectAddFails := func(err error) func(f *fixture) {
		return func(f *fixture) {
			f.useCases.EXPECT().Add(gomock.Any(), gomock.Any()).Return(usecase.CommentView{}, err)
		}
	}
	added := usecase.AddInput{
		ArticleID: articleID,
		AuthorID:  readerID,
		Body:      submittedBody,
	}
	tCases := []struct {
		name   string
		caller *auth.Principal
		body   string
		mock   func(f *fixture)
		want   outcome
	}{
		{
			name:   "a signed-in caller comments as the author",
			caller: reader(),
			body:   body,
			mock:   expectAdded(added),
			want:   outcome{status: 201, body: commentJSON},
		},
		{
			name:   "an article ID in capitals is passed on in its canonical form",
			caller: reader(),
			body:   `{"article_id":"` + strings.ToUpper(articleID) + `","body":"` + submittedBody + `"}`,
			mock:   expectAdded(added),
			want:   outcome{status: 201, body: commentJSON},
		},
		{
			name: "a request with no caller is unauthenticated",
			body: body,
			want: outcome{status: 401, code: "unauthenticated"},
		},
		{
			name:   "a missing article ID is refused before the use cases are called",
			caller: reader(),
			body:   `{"body":"` + submittedBody + `"}`,
			want:   outcome{status: 422, code: "validation_failed", field: "article_id"},
		},
		{
			name:   "an article ID that is not a UUID is refused before the use cases are called",
			caller: reader(),
			body:   `{"article_id":"hello-world","body":"` + submittedBody + `"}`,
			want:   outcome{status: 422, code: "validation_failed", field: "article_id"},
		},
		{
			name:   "a missing body is refused before the use cases are called",
			caller: reader(),
			body:   `{"article_id":"` + articleID + `"}`,
			want:   outcome{status: 422, code: "validation_failed", field: "body"},
		},
		{
			name:   "an unknown field is refused before the use cases are called",
			caller: reader(),
			body:   `{"article_id":"` + articleID + `","body":"` + submittedBody + `","author_id":"x"}`,
			want:   outcome{status: 400, code: "malformed_json"},
		},
		{
			name:   "an article that is not published is not found",
			caller: reader(),
			body:   body,
			mock:   expectAddFails(domain.ErrArticleNotFound),
			want:   outcome{status: 404, code: "not_found"},
		},
		{
			name:   "a caller whose account is gone is not found",
			caller: reader(),
			body:   body,
			mock:   expectAddFails(domain.ErrAuthorNotFound),
			want:   outcome{status: 404, code: "not_found"},
		},
		{
			name:   "a body the rules refuse names the body field without repeating it",
			caller: reader(),
			body:   body,
			mock:   expectAddFails(domain.ErrInvalidBody),
			want:   outcome{status: 422, code: "validation_failed", field: "body"},
		},
		{
			name:   "an unexpected error is an internal error that does not leak its cause",
			caller: reader(),
			body:   body,
			mock:   expectAddFails(errStoreDown),
			want:   outcome{status: 500, code: "internal", logged: "10.0.0.1"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.mock != nil {
				tc.mock(f)
			}

			recorder := f.serve(tc.caller, http.MethodPost, "/v1/comments", tc.body)

			f.check(t, recorder, tc.want)
		})
	}
}
