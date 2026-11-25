//go:build integration

package postgres_test

import (
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// Delete removes a stored comment, and reports one never stored, or a malformed ID, as not found.
func TestCommentRepoDelete(t *testing.T) {
	tCases := []struct {
		name       string
		stored     bool
		commentID  string // "" means the comment's own ID
		wantErr    error
		wantStored bool // the comment is still stored after Delete
	}{
		{name: "a stored comment is removed", stored: true},
		{name: "a comment never stored is not found", wantErr: domain.ErrNotFound},
		{
			name:       "a malformed ID is not found, and nothing is removed",
			stored:     true,
			commentID:  "not-a-uuid",
			wantErr:    domain.ErrNotFound,
			wantStored: true,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newRepo(t)
			comment := commentOn(t, id.New(), id.New(), baseTime)
			if tc.stored {
				store(t, repo, comment)
			}
			target := tc.commentID
			if target == "" {
				target = comment.ID()
			}

			err := repo.Delete(t.Context(), target)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Delete = %v, want %v", err, tc.wantErr)
			}
			_, err = repo.ByID(t.Context(), comment.ID())
			if stored := err == nil; stored != tc.wantStored {
				t.Errorf("stored after Delete = %t (%v), want %t", stored, err, tc.wantStored)
			}
		})
	}
}
