//go:build integration

package postgres_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
)

// List pages through the articles matching a status, newest first, and counts every match.
func TestArticleRepoList(t *testing.T) {
	repo := newRepo(t)
	for i := range 5 {
		article := draft(t, fmt.Sprintf("a%d", i), baseTime.Add(time.Duration(i)*time.Minute))
		if i >= 3 {
			if err := article.Publish(article.CreatedAt()); err != nil {
				t.Fatalf("Publish = %v", err)
			}
		}
		store(t, repo, article)
	}
	tCases := []struct {
		name      string
		q         usecase.ArticleQuery
		wantTotal int
		wantSlugs []string
	}{
		{
			name:      "the first page of published articles holds the newest one",
			q:         usecase.ArticleQuery{Status: domain.StatusPublished, Limit: 1},
			wantTotal: 2,
			wantSlugs: []string{"a4"},
		},
		{
			name:      "the second page of published articles holds the next one",
			q:         usecase.ArticleQuery{Status: domain.StatusPublished, Limit: 1, Offset: 1},
			wantTotal: 2,
			wantSlugs: []string{"a3"},
		},
		{
			name:      "every status is listed newest first",
			q:         usecase.ArticleQuery{Limit: 10},
			wantTotal: 5,
			wantSlugs: []string{"a4", "a3", "a2", "a1", "a0"},
		},
		{
			name:      "an offset past the end returns no articles and still counts them",
			q:         usecase.ArticleQuery{Limit: 10, Offset: 10},
			wantTotal: 5,
			wantSlugs: []string{},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			items, total, err := repo.List(t.Context(), tc.q)

			if err != nil {
				t.Fatalf("List = %v", err)
			}
			slugs := []string{}
			for _, article := range items {
				slugs = append(slugs, article.Slug())
			}
			if total != tc.wantTotal || !reflect.DeepEqual(slugs, tc.wantSlugs) {
				t.Errorf("List = %v (total %d), want %v (total %d)", slugs, total, tc.wantSlugs, tc.wantTotal)
			}
		})
	}
}
