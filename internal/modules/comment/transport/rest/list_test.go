package rest_test

import (
	"net/http"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
)

// List answers 200 with a page of the comments on an article, to anyone. It refuses an article_id that is
// missing or not a UUID, an author_id that is not a UUID, and a limit or offset that is not a whole number of 0
// or more, with invalid_query, never echoing the value; the comments of an article that is not published are
// not found.
func TestHandlerList(t *testing.T) {
	const page = `{"items":[` + commentJSON + `],"total":11,"limit":5,"offset":10}`
	expectListed := func(input usecase.ListInput, output usecase.ListOutput) func(f *fixture) {
		return func(f *fixture) { f.useCases.EXPECT().List(gomock.Any(), input).Return(output, nil) }
	}
	tCases := []struct {
		name        string
		query       string
		mock        func(f *fixture)
		want        outcome
		mustNotEcho string // query value the body must not contain
	}{
		{
			name:  "the comments on an article are listed, with the page passed on",
			query: "?article_id=" + articleID + "&limit=5&offset=10",
			mock: expectListed(usecase.ListInput{
				ArticleID: articleID,
				Limit:     5,
				Offset:    10,
			}, usecase.ListOutput{
				Items:  []usecase.CommentView{viewFixture()},
				Total:  11,
				Limit:  5,
				Offset: 10,
			}),
			want: outcome{status: 200, body: page},
		},
		{
			name:  "the author filter is passed on, both IDs in their canonical form",
			query: "?article_id=" + strings.ToUpper(articleID) + "&author_id=" + strings.ToUpper(readerID),
			mock: expectListed(usecase.ListInput{ArticleID: articleID, AuthorID: readerID},
				usecase.ListOutput{Limit: 20}),
			want: outcome{status: 200, body: `{"items":[],"total":0,"limit":20,"offset":0}`},
		},
		{
			name:  "a missing article ID is refused",
			query: "?limit=5",
			want:  outcome{status: 400, code: "invalid_query"},
		},
		{
			name:        "an article ID that is not a UUID is refused without being echoed",
			query:       "?article_id=hello-world",
			want:        outcome{status: 400, code: "invalid_query"},
			mustNotEcho: "hello-world",
		},
		{
			name:        "an author ID that is not a UUID is refused without being echoed",
			query:       "?article_id=" + articleID + "&author_id=ada",
			want:        outcome{status: 400, code: "invalid_query"},
			mustNotEcho: "ada",
		},
		{
			name:  "a negative limit is refused",
			query: "?article_id=" + articleID + "&limit=-1",
			want:  outcome{status: 400, code: "invalid_query"},
		},
		{
			name:        "an offset that is not a number is refused without being echoed",
			query:       "?article_id=" + articleID + "&offset=lots",
			want:        outcome{status: 400, code: "invalid_query"},
			mustNotEcho: "lots",
		},
		{
			name:  "the comments of an article that is not published are not found",
			query: "?article_id=" + articleID,
			mock: func(f *fixture) {
				f.useCases.EXPECT().List(gomock.Any(), gomock.Any()).Return(usecase.ListOutput{}, domain.ErrArticleNotFound)
			},
			want: outcome{status: 404, code: "not_found"},
		},
		{
			name:  "an unexpected error is an internal error that does not leak its cause",
			query: "?article_id=" + articleID,
			mock: func(f *fixture) {
				f.useCases.EXPECT().List(gomock.Any(), gomock.Any()).Return(usecase.ListOutput{}, errStoreDown)
			},
			want: outcome{status: 500, code: "internal", logged: "10.0.0.1"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.mock != nil {
				tc.mock(f)
			}

			recorder := f.serve(nil, http.MethodGet, "/v1/comments"+tc.query, "")

			f.check(t, recorder, tc.want)
			if tc.mustNotEcho != "" && strings.Contains(recorder.Body.String(), tc.mustNotEcho) {
				t.Errorf("body = %s, want %q not echoed", recorder.Body, tc.mustNotEcho)
			}
		})
	}
}
