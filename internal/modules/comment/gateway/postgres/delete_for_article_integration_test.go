//go:build integration

package postgres_test

import (
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// DeleteForArticle removes every comment on one article and leaves the comments on the others. An article with
// no comments left is not an error, so calling it again changes nothing.
func TestCommentRepoDeleteForArticle(t *testing.T) {
	tCases := []struct {
		name  string
		calls int
	}{
		{name: "the article's comments are removed and the others stay", calls: 1},
		{name: "a second call changes nothing and is not an error", calls: 2},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newRepo(t)
			articleID, otherArticleID := id.New(), id.New()
			removed := []*domain.Comment{
				commentOn(t, articleID, id.New(), baseTime),
				commentOn(t, articleID, id.New(), baseTime),
			}
			kept := commentOn(t, otherArticleID, id.New(), baseTime)
			for _, comment := range append(removed, kept) {
				store(t, repo, comment)
			}

			for call := range tc.calls {
				if err := repo.DeleteForArticle(t.Context(), articleID); err != nil {
					t.Fatalf("DeleteForArticle call %d = %v, want nil", call+1, err)
				}
			}

			for _, comment := range removed {
				if _, err := repo.ByID(t.Context(), comment.ID()); !errors.Is(err, domain.ErrNotFound) {
					t.Errorf("ByID of a removed comment = %v, want %v", err, domain.ErrNotFound)
				}
			}
			if _, err := repo.ByID(t.Context(), kept.ID()); err != nil {
				t.Errorf("ByID of a comment on another article = %v, want it kept", err)
			}
		})
	}
}
