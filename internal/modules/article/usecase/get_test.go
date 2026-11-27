package usecase_test

import (
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// Get returns the article with the given ID when it has the status asked for, any status
// when none is asked for, and domain.ErrNotFound otherwise.
func TestGet(t *testing.T) {
	tCases := []struct {
		name    string
		status  domain.Status
		found   *domain.Article
		findErr error
		want    usecase.ArticleView
		wantErr error
	}{
		{
			name:   "a draft is returned when drafts are asked for",
			status: domain.StatusDraft,
			found:  articleFixture(),
			want:   viewFixture(),
		},
		{
			name:  "a published article is returned when any status is asked for",
			found: publishedFixture(),
			want:  publishedView(),
		},
		{
			name:    "a draft asked for as published is not found",
			status:  domain.StatusPublished,
			found:   articleFixture(),
			wantErr: domain.ErrNotFound,
		},
		{
			name:    "an unknown article is not found",
			status:  domain.StatusPublished,
			findErr: domain.ErrNotFound,
			wantErr: domain.ErrNotFound,
		},
		{name: "the store's failure is returned", findErr: errStoreDown, wantErr: errStoreDown},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.repo.EXPECT().ByID(gomock.Any(), articleID).Return(tc.found, tc.findErr)

			got, err := f.useCase.Get(t.Context(), articleID, tc.status)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Get = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Get = %+v, want %+v", got, tc.want)
			}
		})
	}
}
