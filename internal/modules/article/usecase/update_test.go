package usecase_test

import (
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// Update changes only the fields given, through the store's update, and refuses a new slug for a published
// article.
func TestUpdate(t *testing.T) {
	tCases := []struct {
		name    string
		in      usecase.UpdateInput
		mock    func(f *fixture)
		want    usecase.ArticleView
		wantErr error
	}{
		{
			name: "only the given title changes, and the article is saved",
			in: usecase.UpdateInput{
				ID:    articleID,
				Title: new("New"),
			},
			mock: func(f *fixture) { f.expectUpdate(articleFixture()) },
			want: viewWith(func(view *usecase.ArticleView) { view.Title = "New" }),
		},
		{
			name: "an unknown article is not found",
			in: usecase.UpdateInput{
				ID:    articleID,
				Title: new("New"),
			},
			mock: func(f *fixture) {
				f.repo.EXPECT().Update(gomock.Any(), articleID, gomock.Any()).Return(nil, domain.ErrNotFound)
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "a published article keeps its slug",
			in: usecase.UpdateInput{
				ID:   articleID,
				Slug: new("renamed"),
			},
			mock:    func(f *fixture) { f.expectUpdate(publishedFixture()) },
			wantErr: domain.ErrSlugLocked,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			got, err := f.useCase.Update(t.Context(), tc.in)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Update = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Update = %+v, want %+v", got, tc.want)
			}
		})
	}
}
