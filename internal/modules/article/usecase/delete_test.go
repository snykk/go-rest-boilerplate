package usecase_test

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// Delete removes the article with the given ID and returns the store's error.
func TestDelete(t *testing.T) {
	tCases := []struct {
		name     string
		storeErr error
		wantErr  error
	}{
		{name: "a stored article is removed"},
		{name: "an unknown article is not found", storeErr: domain.ErrNotFound, wantErr: domain.ErrNotFound},
		{name: "a store failure is returned", storeErr: errStoreDown, wantErr: errStoreDown},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.repo.EXPECT().Delete(gomock.Any(), articleID).Return(tc.storeErr)

			if err := f.useCase.Delete(t.Context(), articleID); !errors.Is(err, tc.wantErr) {
				t.Errorf("Delete = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
