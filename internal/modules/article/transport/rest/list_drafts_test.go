package rest_test

import (
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// ListDrafts asks for drafts only, whatever status the query names, with a
// valid limit and offset, and refuses any other limit or offset with invalid_query.
func TestHandlerListDrafts(t *testing.T) {
	tCases := []struct {
		name       string
		query      string
		mock       func(f *fixture)
		wantStatus int
		wantBody   string // text the body contains
	}{
		{
			name:  "drafts are listed, with the limit and offset passed on",
			query: "?limit=5&offset=10",
			mock: func(f *fixture) {
				f.useCases.EXPECT().List(gomock.Any(), usecase.ListInput{
					Status: domain.StatusDraft,
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
			name:  "a status in the query is ignored: only drafts are listed",
			query: "?status=published",
			mock: func(f *fixture) {
				f.useCases.EXPECT().List(gomock.Any(), usecase.ListInput{Status: domain.StatusDraft}).
					Return(usecase.ListOutput{}, nil)
			},
			wantStatus: 200,
			wantBody:   `"items":[]`,
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

			recorder := f.serve("GET", "/v1/articles/drafts"+tc.query, "")

			if recorder.Code != tc.wantStatus || !strings.Contains(recorder.Body.String(), tc.wantBody) {
				t.Errorf("listDrafts = %d %s, want %d with %s", recorder.Code, recorder.Body, tc.wantStatus, tc.wantBody)
			}
		})
	}
}
