package usecase_test

import (
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
)

// Add stores a comment on a published article with its author's display name copied in, and refuses an
// article that is not published, an author who is gone and a body outside the rules without storing anything.
func TestCommentAdd(t *testing.T) {
	input := usecase.AddInput{
		ArticleID: articleID,
		AuthorID:  authorID,
		Body:      " Nice one. ",
	}
	expectAuthor := func(f *fixture) {
		f.expectPublished(articleID, true)
		f.authors.EXPECT().DisplayName(gomock.Any(), authorID).Return("Ada", nil)
	}
	tCases := []struct {
		name    string
		input   usecase.AddInput
		mock    func(f *fixture)
		want    usecase.CommentView
		wantErr error
	}{
		{
			name:  "a comment on a published article is stored with the author's name",
			input: input,
			mock: func(f *fixture) {
				expectAuthor(f)
				f.comments.EXPECT().Create(gomock.Any(), commentFixture()).Return(nil)
			},
			want: viewFixture(),
		},
		{
			name:    "a comment on a draft is not found",
			input:   input,
			mock:    func(f *fixture) { f.expectPublished(articleID, false) },
			wantErr: domain.ErrArticleNotFound,
		},
		{
			name:  "an article module that cannot tell fails the call",
			input: input,
			mock: func(f *fixture) {
				f.articles.EXPECT().Published(gomock.Any(), articleID).Return(false, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:  "an author who is gone is not found",
			input: input,
			mock: func(f *fixture) {
				f.expectPublished(articleID, true)
				f.authors.EXPECT().DisplayName(gomock.Any(), authorID).Return("", domain.ErrAuthorNotFound)
			},
			wantErr: domain.ErrAuthorNotFound,
		},
		{
			name: "a blank body is refused before the store is asked",
			input: usecase.AddInput{
				ArticleID: articleID,
				AuthorID:  authorID,
				Body:      "  ",
			},
			mock:    expectAuthor,
			wantErr: domain.ErrInvalidBody,
		},
		{
			name:  "a store that cannot be reached fails the call",
			input: input,
			mock: func(f *fixture) {
				expectAuthor(f)
				f.comments.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errStoreDown)
			},
			wantErr: errStoreDown,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			got, err := f.useCase.Add(t.Context(), tc.input)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Add = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Add = %+v, want %+v", got, tc.want)
			}
		})
	}
}
