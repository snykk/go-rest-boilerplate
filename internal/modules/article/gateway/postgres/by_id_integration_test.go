//go:build integration

package postgres_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// ByID returns a stored article as it was stored, and reports an unknown or malformed ID as not found.
func TestArticleRepoByID(t *testing.T) {
	repo := newRepo(t)
	stored := draft(t, "hello", baseTime)
	store(t, repo, stored)
	tCases := []struct {
		name      string
		articleID string
		want      *domain.Article
		wantErr   error
	}{
		{name: "a stored article is returned as it was stored", articleID: stored.ID(), want: stored},
		{name: "an unknown ID is not found", articleID: id.New(), wantErr: domain.ErrNotFound},
		{name: "a malformed ID is not found", articleID: "not-a-uuid", wantErr: domain.ErrNotFound},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.ByID(t.Context(), tc.articleID)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ByID = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ByID = %+v, want %+v", got, tc.want)
			}
		})
	}
}
