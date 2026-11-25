package usecase_test

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

// DeleteForArticle removes every comment on the article, whatever its status, and returns the store's failure.
func TestCommentDeleteForArticle(t *testing.T) {
	tCases := []struct {
		name     string
		storeErr error
		wantErr  error
	}{
		{name: "the article's comments are removed"},
		{name: "a store that cannot be reached fails the call", storeErr: errStoreDown, wantErr: errStoreDown},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.comments.EXPECT().DeleteForArticle(gomock.Any(), articleID).Return(tc.storeErr)

			if err := f.useCase.DeleteForArticle(t.Context(), articleID); !errors.Is(err, tc.wantErr) {
				t.Errorf("DeleteForArticle = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
