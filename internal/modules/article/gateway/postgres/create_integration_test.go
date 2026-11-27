//go:build integration

package postgres_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
)

// Create stores an article once, and lets exactly one of several concurrent creations of a slug through.
func TestArticleRepoCreate(t *testing.T) {
	tCases := []struct {
		name        string
		existing    bool // an article with the same slug is stored first
		concurrent  int  // creations of the same slug, run at the same time
		wantCreated int32
		wantTaken   int32
	}{
		{name: "a new slug is stored", concurrent: 1, wantCreated: 1},
		{name: "a slug another article uses is refused", existing: true, concurrent: 1, wantTaken: 1},
		{
			name:        "of ten creations of one slug at the same time exactly one is stored",
			concurrent:  10,
			wantCreated: 1,
			wantTaken:   9,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newRepo(t)
			if tc.existing {
				store(t, repo, draft(t, "same", baseTime))
			}
			articles := make([]*domain.Article, tc.concurrent)
			for i := range articles {
				articles[i] = draft(t, "same", baseTime)
			}
			var created, taken atomic.Int32
			var running sync.WaitGroup

			for _, article := range articles {
				running.Go(func() {
					switch err := repo.Create(t.Context(), article); {
					case err == nil:
						created.Add(1)
					case errors.Is(err, domain.ErrSlugTaken):
						taken.Add(1)
					}
				})
			}
			running.Wait()

			if created.Load() != tc.wantCreated || taken.Load() != tc.wantTaken {
				t.Errorf("created %d, taken %d; want %d, %d", created.Load(), taken.Load(), tc.wantCreated, tc.wantTaken)
			}
		})
	}
}
