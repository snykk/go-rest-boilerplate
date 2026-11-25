//go:build integration

package postgres_test

import (
	"reflect"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// Create stores a comment as it was written. Its article and author are plain IDs: no row of the article or the
// identity module needs to exist for it.
func TestCommentRepoCreate(t *testing.T) {
	tCases := []struct {
		name string
	}{
		{name: "a comment whose article and author have no row here is stored as it was written"},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newRepo(t)
			comment := commentOn(t, id.New(), id.New(), baseTime)

			if err := repo.Create(t.Context(), comment); err != nil {
				t.Fatalf("Create = %v", err)
			}

			got, err := repo.ByID(t.Context(), comment.ID())
			if err != nil || !reflect.DeepEqual(got, comment) {
				t.Errorf("ByID after Create = %+v, %v; want %+v", got, err, comment)
			}
		})
	}
}
