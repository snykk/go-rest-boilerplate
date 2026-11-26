package usecase_test

import (
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// List asks the store for a page within the use case's page sizes (20 by default and at most 100 in the
// fixture), and returns what the store finds.
func TestList(t *testing.T) {
	tCases := []struct {
		name      string
		pageSize  usecase.PageSize // zero means the fixture's
		in        usecase.ListInput
		wantQuery usecase.ArticleQuery
		found     []domain.Article
		total     int
		storeErr  error
		want      usecase.ListOutput
		wantErr   error
	}{
		{
			name:      "an empty request asks for the first 20 articles of any status",
			wantQuery: usecase.ArticleQuery{Limit: 20},
			want:      usecase.ListOutput{Limit: 20},
		},
		{
			name: "a limit over 100 and a negative offset are clamped",
			in: usecase.ListInput{
				Limit:  1000,
				Offset: -5,
			},
			wantQuery: usecase.ArticleQuery{Limit: 100},
			want:      usecase.ListOutput{Limit: 100},
		},
		{
			name: "the status and page asked for are passed on, with what the store finds",
			in: usecase.ListInput{
				Status: domain.StatusPublished,
				Limit:  5,
				Offset: 10,
			},
			wantQuery: usecase.ArticleQuery{Status: domain.StatusPublished, Limit: 5, Offset: 10},
			found:     []domain.Article{*publishedFixture()},
			total:     11,
			want: usecase.ListOutput{
				Items:  []usecase.ArticleView{publishedView()},
				Total:  11,
				Limit:  5,
				Offset: 10,
			},
		},
		{
			name:      "a store that cannot be reached fails the call",
			wantQuery: usecase.ArticleQuery{Limit: 20},
			storeErr:  errStoreDown,
			wantErr:   errStoreDown,
		},
		{
			name:      "a use case with other page sizes uses its own default",
			pageSize:  usecase.PageSize{Default: 5, Max: 10},
			wantQuery: usecase.ArticleQuery{Limit: 5},
			want:      usecase.ListOutput{Limit: 5},
		},
		{
			name:      "a use case with other page sizes caps a page at its own maximum",
			pageSize:  usecase.PageSize{Default: 5, Max: 10},
			in:        usecase.ListInput{Limit: 50},
			wantQuery: usecase.ArticleQuery{Limit: 10},
			want:      usecase.ListOutput{Limit: 10},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.pageSize != (usecase.PageSize{}) {
				f.useCase = usecase.NewArticleUsecase(usecase.ArticleDeps{
					Articles: f.repo,
					Clock:    fixedClock{},
					IDs:      fixedID{},
					PageSize: tc.pageSize,
				})
			}
			f.repo.EXPECT().List(gomock.Any(), tc.wantQuery).Return(tc.found, tc.total, tc.storeErr)

			got, err := f.useCase.List(t.Context(), tc.in)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("List = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("List = %+v, want %+v", got, tc.want)
			}
		})
	}
}
