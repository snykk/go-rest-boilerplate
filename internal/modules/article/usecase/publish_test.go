package usecase_test

import (
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// Publish publishes a stored draft at the current time, through the store's update.
func TestPublish(t *testing.T) {
	tCases := []struct {
		name    string
		mock    func(f *fixture)
		want    usecase.ArticleView
		wantErr error
	}{
		{
			name: "a draft with a body is published at the current time",
			mock: func(f *fixture) { f.expectUpdate(articleFixture()) },
			want: publishedView(),
		},
		{
			name:    "a published article is not published again",
			mock:    func(f *fixture) { f.expectUpdate(publishedFixture()) },
			wantErr: domain.ErrAlreadyPublished,
		},
		{
			name: "an unknown article is not found",
			mock: func(f *fixture) {
				f.repo.EXPECT().Update(gomock.Any(), articleID, gomock.Any()).Return(nil, domain.ErrNotFound)
			},
			wantErr: domain.ErrNotFound,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			got, err := f.useCase.Publish(t.Context(), articleID)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Publish = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Publish = %+v, want %+v", got, tc.want)
			}
		})
	}
}
