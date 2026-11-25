package usecase_test

import (
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
)

// List returns a page of the comments on a published article, within the use case's page sizes (20 by default
// and at most 100 in the fixture). The comments of an article that is not published, a draft or one that never
// existed, are not found rather than an empty page.
func TestCommentList(t *testing.T) {
	const unknownID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c99"
	expectPage := func(query usecase.CommentQuery, found []domain.Comment, total int) func(f *fixture) {
		return func(f *fixture) {
			f.expectPublished(query.ArticleID, true)
			f.comments.EXPECT().List(gomock.Any(), query).Return(found, total, nil)
		}
	}
	tCases := []struct {
		name     string
		pageSize usecase.PageSize // zero means the fixture's
		input    usecase.ListInput
		mock     func(f *fixture)
		want     usecase.ListOutput
		wantErr  error
	}{
		{
			name:    "a draft's comments are not found",
			input:   usecase.ListInput{ArticleID: articleID},
			mock:    func(f *fixture) { f.expectPublished(articleID, false) },
			wantErr: domain.ErrArticleNotFound,
		},
		{
			name:    "an article that never existed is not found",
			input:   usecase.ListInput{ArticleID: unknownID},
			mock:    func(f *fixture) { f.expectPublished(unknownID, false) },
			wantErr: domain.ErrArticleNotFound,
		},
		{
			name:  "an article module that cannot tell fails the call",
			input: usecase.ListInput{ArticleID: articleID},
			mock: func(f *fixture) {
				f.articles.EXPECT().Published(gomock.Any(), articleID).Return(false, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:  "an empty page asks for the first 20 comments on the article",
			input: usecase.ListInput{ArticleID: articleID},
			mock:  expectPage(usecase.CommentQuery{ArticleID: articleID, Limit: 20}, nil, 0),
			want:  usecase.ListOutput{Limit: 20},
		},
		{
			name: "a limit over 100 and a negative offset are clamped",
			input: usecase.ListInput{
				ArticleID: articleID,
				Limit:     1000,
				Offset:    -5,
			},
			mock: expectPage(usecase.CommentQuery{ArticleID: articleID, Limit: 100}, nil, 0),
			want: usecase.ListOutput{Limit: 100},
		},
		{
			name:     "a use case with other page sizes uses its own default",
			pageSize: usecase.PageSize{Default: 5, Max: 10},
			input:    usecase.ListInput{ArticleID: articleID},
			mock:     expectPage(usecase.CommentQuery{ArticleID: articleID, Limit: 5}, nil, 0),
			want:     usecase.ListOutput{Limit: 5},
		},
		{
			name:     "a use case with other page sizes caps a page at its own maximum",
			pageSize: usecase.PageSize{Default: 5, Max: 10},
			input:    usecase.ListInput{ArticleID: articleID, Limit: 50},
			mock:     expectPage(usecase.CommentQuery{ArticleID: articleID, Limit: 10}, nil, 0),
			want:     usecase.ListOutput{Limit: 10},
		},
		{
			name: "the author filter is passed on",
			input: usecase.ListInput{
				ArticleID: articleID,
				AuthorID:  authorID,
				Limit:     5,
				Offset:    10,
			},
			mock: expectPage(usecase.CommentQuery{
				ArticleID: articleID,
				AuthorID:  authorID,
				Limit:     5,
				Offset:    10,
			}, []domain.Comment{*commentFixture()}, 11),
			want: usecase.ListOutput{
				Items:  []usecase.CommentView{viewFixture()},
				Total:  11,
				Limit:  5,
				Offset: 10,
			},
		},
		{
			name:  "a store that cannot be reached fails the call",
			input: usecase.ListInput{ArticleID: articleID},
			mock: func(f *fixture) {
				f.expectPublished(articleID, true)
				f.comments.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, 0, errStoreDown)
			},
			wantErr: errStoreDown,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.pageSize != (usecase.PageSize{}) {
				f.withPageSize(tc.pageSize)
			}
			tc.mock(f)

			got, err := f.useCase.List(t.Context(), tc.input)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("List = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("List = %+v, want %+v", got, tc.want)
			}
		})
	}
}
