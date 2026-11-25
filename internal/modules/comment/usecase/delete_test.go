package usecase_test

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
)

// Delete removes a comment for its author or for an editor, and refuses anyone else without removing it.
func TestCommentDelete(t *testing.T) {
	expectDeleted := func(f *fixture) {
		f.expectFound(commentFixture())
		f.comments.EXPECT().Delete(gomock.Any(), commentID).Return(nil)
	}
	tCases := []struct {
		name    string
		input   usecase.DeleteInput
		mock    func(f *fixture)
		wantErr error
	}{
		{
			name:  "the author deletes",
			input: usecase.DeleteInput{ID: commentID, CallerID: authorID},
			mock:  expectDeleted,
		},
		{
			name: "an editor deletes another's",
			input: usecase.DeleteInput{
				ID:             commentID,
				CallerID:       otherID,
				CallerIsEditor: true,
			},
			mock: expectDeleted,
		},
		{
			name:    "a reader deleting another's is forbidden",
			input:   usecase.DeleteInput{ID: commentID, CallerID: otherID},
			mock:    func(f *fixture) { f.expectFound(commentFixture()) },
			wantErr: domain.ErrForbidden,
		},
		{
			name:  "an unknown comment is not found",
			input: usecase.DeleteInput{ID: commentID, CallerID: authorID},
			mock: func(f *fixture) {
				f.comments.EXPECT().ByID(gomock.Any(), commentID).Return(nil, domain.ErrNotFound)
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name:  "a comment deleted meanwhile is not found",
			input: usecase.DeleteInput{ID: commentID, CallerID: authorID},
			mock: func(f *fixture) {
				f.expectFound(commentFixture())
				f.comments.EXPECT().Delete(gomock.Any(), commentID).Return(domain.ErrNotFound)
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name:  "a store that cannot be reached fails the call",
			input: usecase.DeleteInput{ID: commentID, CallerID: authorID},
			mock: func(f *fixture) {
				f.comments.EXPECT().ByID(gomock.Any(), commentID).Return(nil, errStoreDown)
			},
			wantErr: errStoreDown,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			if err := f.useCase.Delete(t.Context(), tc.input); !errors.Is(err, tc.wantErr) {
				t.Errorf("Delete = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
