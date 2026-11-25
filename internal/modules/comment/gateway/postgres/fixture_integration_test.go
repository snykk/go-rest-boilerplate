//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/gateway/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
	"github.com/snykk/go-rest-boilerplate/internal/testenv"
	"github.com/snykk/go-rest-boilerplate/migrations"
)

var baseTime = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC) // whole seconds: Postgres keeps microseconds

// newRepo is a repository over a database of its own, migrated with the real migrations.
func newRepo(t *testing.T) *postgres.CommentRepo {
	t.Helper()
	return postgres.NewCommentRepo(testenv.Postgres(t, migrations.FS))
}

// commentOn is a valid comment on the article with articleID by the user with authorID, created at createdAt
// and not stored.
func commentOn(t *testing.T, articleID, authorID string, createdAt time.Time) *domain.Comment {
	t.Helper()
	comment, err := domain.New(id.New(), articleID, authorID, "Ada", "Nice one.", createdAt)
	if err != nil {
		t.Fatalf("domain.New = %v", err)
	}
	return comment
}

// store creates comment in repo, failing the test when it cannot.
func store(t *testing.T, repo *postgres.CommentRepo, comment *domain.Comment) {
	t.Helper()
	if err := repo.Create(t.Context(), comment); err != nil {
		t.Fatalf("Create = %v", err)
	}
}
