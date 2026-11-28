//go:build integration

package postgres_test

import (
	"errors"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// Delete removes a stored article, and reports one never stored as not found.
func TestArticleRepoDelete(t *testing.T) {
	tCases := []struct {
		name    string
		stored  bool
		wantErr error
	}{
		{name: "a stored article is removed", stored: true},
		{name: "an article never stored is not found", wantErr: domain.ErrNotFound},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newRepo(t)
			article := draft(t, "a", baseTime)
			if tc.stored {
				store(t, repo, article)
			}

			err := repo.Delete(t.Context(), article.ID())

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Delete = %v, want %v", err, tc.wantErr)
			}
			if _, err := repo.ByID(t.Context(), article.ID()); !errors.Is(err, domain.ErrNotFound) {
				t.Errorf("ByID after Delete = %v, want %v", err, domain.ErrNotFound)
			}
		})
	}
}
