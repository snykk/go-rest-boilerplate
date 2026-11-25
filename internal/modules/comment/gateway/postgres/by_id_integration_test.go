//go:build integration

package postgres_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// ByID returns a stored comment as it was stored, and reports an unknown or malformed ID as not found.
func TestCommentRepoByID(t *testing.T) {
	repo := newRepo(t)
	stored := commentOn(t, id.New(), id.New(), baseTime)
	store(t, repo, stored)
	tCases := []struct {
		name      string
		commentID string
		want      *domain.Comment
		wantErr   error
	}{
		{name: "a stored comment is returned as it was stored", commentID: stored.ID(), want: stored},
		{name: "an unknown ID is not found", commentID: id.New(), wantErr: domain.ErrNotFound},
		{name: "a malformed ID is not found", commentID: "not-a-uuid", wantErr: domain.ErrNotFound},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.ByID(t.Context(), tc.commentID)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ByID = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ByID = %+v, want %+v", got, tc.want)
			}
		})
	}
}
