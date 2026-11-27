package rest_test

import (
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// List asks for published articles only, whatever status the query names, with a
// valid limit and offset. Any other limit or offset is refused with invalid_query, and the value is never
// echoed.
func TestHandlerList(t *testing.T) {
	tCases := []struct {
		name        string
		query       string
		mock        func(f *fixture)
		wantStatus  int
		wantBody    string // text the body contains
		mustNotEcho string // query value the body must not contain
	}{
		{
			name:  "only published articles are listed, with the limit and offset passed on",
			query: "?limit=5&offset=10",
			mock: func(f *fixture) {
				f.useCases.EXPECT().List(gomock.Any(), usecase.ListInput{
					Status: domain.StatusPublished,
					Limit:  5,
					Offset: 10,
				}).Return(usecase.ListOutput{
					Items:  []usecase.ArticleView{viewFixture()},
					Total:  1,
					Limit:  5,
					Offset: 10,
				}, nil)
			},
			wantStatus: 200,
			wantBody:   `"total":1`,
		},
		{
			name: "an empty list is an empty array",
			mock: func(f *fixture) {
				f.useCases.EXPECT().List(gomock.Any(), usecase.ListInput{Status: domain.StatusPublished}).
					Return(usecase.ListOutput{}, nil)
			},
			wantStatus: 200,
			wantBody:   `"items":[]`,
		},
		{
			name:  "a status in the query is ignored: only published articles are listed",
			query: "?status=draft",
			mock: func(f *fixture) {
				f.useCases.EXPECT().List(gomock.Any(), usecase.ListInput{Status: domain.StatusPublished}).
					Return(usecase.ListOutput{}, nil)
			},
			wantStatus: 200,
			wantBody:   `"items":[]`,
		},
		{
			name:        "a limit that is not a number is refused without being echoed",
			query:       "?limit=lots",
			mock:        func(*fixture) {},
			wantStatus:  400,
			wantBody:    `"code":"invalid_query"`,
			mustNotEcho: "lots",
		},
		{
			name:       "a negative limit is refused",
			query:      "?limit=-1",
			mock:       func(*fixture) {},
			wantStatus: 400,
			wantBody:   `"code":"invalid_query"`,
		},
		{
			name:       "a limit that is not a number is refused",
			query:      "?limit=abc",
			mock:       func(*fixture) {},
			wantStatus: 400,
			wantBody:   `"code":"invalid_query"`,
		},
		{
			name:       "a negative offset is refused",
			query:      "?offset=-3",
			mock:       func(*fixture) {},
			wantStatus: 400,
			wantBody:   `"code":"invalid_query"`,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			recorder := f.serve("GET", "/v1/articles"+tc.query, "")

			if recorder.Code != tc.wantStatus || !strings.Contains(recorder.Body.String(), tc.wantBody) {
				t.Errorf("List = %d %s, want %d with %s", recorder.Code, recorder.Body, tc.wantStatus, tc.wantBody)
			}
			if tc.mustNotEcho != "" && strings.Contains(recorder.Body.String(), tc.mustNotEcho) {
				t.Errorf("body = %s, want %q not echoed", recorder.Body, tc.mustNotEcho)
			}
		})
	}
}
