//go:build integration

package postgres_test

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// Update saves the change it applies to a stored article and returns the result. It saves nothing when the
// change fails or takes another article's slug, and reports an unknown or malformed ID as not found.
func TestArticleRepoUpdate(t *testing.T) {
	later := baseTime.Add(time.Hour)
	errRefused := errors.New("change refused")
	storedID := func(article *domain.Article) string { return article.ID() }
	mustNotRun := func(*domain.Article) error { return errors.New("the change ran") }
	tCases := []struct {
		name    string
		id      func(stored *domain.Article) string
		change  func(article *domain.Article) error
		wantErr error
		want    func(article *domain.Article) // what the stored article went through; nil means nothing
	}{
		{
			name:   "a change to a stored article is saved and returned",
			id:     storedID,
			change: func(article *domain.Article) error { return article.Publish(later) },
			want:   func(article *domain.Article) { _ = article.Publish(later) },
		},
		{
			name: "a change that fails saves nothing",
			id:   storedID,
			change: func(article *domain.Article) error {
				_ = article.Edit(nil, new("changed"), nil, later)
				return errRefused
			},
			wantErr: errRefused,
		},
		{
			name:    "a new slug another article uses is refused",
			id:      storedID,
			change:  func(article *domain.Article) error { return article.Edit(new("taken"), nil, nil, later) },
			wantErr: domain.ErrSlugTaken,
		},
		{
			name:    "an article never stored is not found",
			id:      func(*domain.Article) string { return id.New() },
			change:  mustNotRun,
			wantErr: domain.ErrNotFound,
		},
		{
			name:    "a malformed ID is not found",
			id:      func(*domain.Article) string { return "not-a-uuid" },
			change:  mustNotRun,
			wantErr: domain.ErrNotFound,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newRepo(t)
			stored := draft(t, "a", baseTime)
			store(t, repo, stored)
			store(t, repo, draft(t, "taken", baseTime))

			got, err := repo.Update(t.Context(), tc.id(stored), tc.change)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Update = %v, want %v", err, tc.wantErr)
			}
			want := *stored
			if tc.want != nil {
				tc.want(&want)
			}
			read, err := repo.ByID(t.Context(), stored.ID())
			if err != nil || !reflect.DeepEqual(read, &want) {
				t.Errorf("ByID after Update = %+v, %v; want %+v", read, err, want)
			}
			if tc.wantErr == nil && !reflect.DeepEqual(got, read) {
				t.Errorf("Update = %+v, want what is stored: %+v", got, read)
			}
		})
	}
}

// Update applies concurrent changes to one article one after the other, so none of them is lost.
func TestArticleRepoUpdateConcurrent(t *testing.T) {
	repo := newRepo(t)
	stored := draft(t, "a", baseTime)
	store(t, repo, stored)
	var running sync.WaitGroup

	for range 10 {
		running.Go(func() {
			_, err := repo.Update(t.Context(), stored.ID(), func(article *domain.Article) error {
				return article.Edit(nil, new(article.Title()+"!"), nil, baseTime)
			})
			if err != nil {
				t.Errorf("Update = %v", err)
			}
		})
	}
	running.Wait()

	got, err := repo.ByID(t.Context(), stored.ID())
	if err != nil {
		t.Fatalf("ByID = %v", err)
	}
	if want := stored.Title() + strings.Repeat("!", 10); got.Title() != want {
		t.Errorf("title = %q, want %q: concurrent changes were lost", got.Title(), want)
	}
}
