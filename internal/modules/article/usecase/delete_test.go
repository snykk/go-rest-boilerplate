package usecase_test

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/contract"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// Delete removes the article with the given ID and announces it once, and announces
// nothing when the store removed nothing.
func TestDelete(t *testing.T) {
	tCases := []struct {
		name      string
		storeErr  error
		wantEvent bool
		wantErr   error
	}{
		{name: "a removed article is announced once", wantEvent: true},
		{
			name:     "an unknown article is not found and announces nothing",
			storeErr: domain.ErrNotFound,
			wantErr:  domain.ErrNotFound,
		},
		{name: "a store failure announces nothing", storeErr: errStoreDown, wantErr: errStoreDown},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.repo.EXPECT().Delete(gomock.Any(), articleID).Return(tc.storeErr)
			if tc.wantEvent {
				f.events.EXPECT().Deleted(gomock.Any(), contract.ArticleDeleted{
					ID: articleID,
					At: baseTime,
				}).Times(1)
			}

			if err := f.useCase.Delete(t.Context(), articleID); !errors.Is(err, tc.wantErr) {
				t.Errorf("Delete = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
