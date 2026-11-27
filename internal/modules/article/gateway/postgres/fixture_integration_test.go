//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/gateway/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
	"github.com/snykk/go-rest-boilerplate/internal/testenv"
	"github.com/snykk/go-rest-boilerplate/migrations"
)

var baseTime = time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC) // whole seconds: Postgres keeps microseconds

// newRepo is a repository over a database of its own, migrated with the real migrations.
func newRepo(t *testing.T) *postgres.ArticleRepo {
	t.Helper()
	return postgres.NewArticleRepo(testenv.Postgres(t, migrations.FS))
}

// draft is a valid draft with the given slug, created at createdAt.
func draft(t *testing.T, slug string, createdAt time.Time) *domain.Article {
	t.Helper()
	article, err := domain.New(id.New(), slug, "Title "+slug, "body", createdAt)
	if err != nil {
		t.Fatalf("domain.New = %v", err)
	}
	return article
}

// store creates article in repo, failing the test when it cannot.
func store(t *testing.T, repo *postgres.ArticleRepo, article *domain.Article) {
	t.Helper()
	if err := repo.Create(t.Context(), article); err != nil {
		t.Fatalf("Create = %v", err)
	}
}
