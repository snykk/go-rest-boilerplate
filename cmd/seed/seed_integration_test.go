//go:build integration

package main

import (
	"log/slog"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/app"
	"github.com/snykk/go-rest-boilerplate/internal/platform/mailer"
	"github.com/snykk/go-rest-boilerplate/internal/platform/telemetry"
	"github.com/snykk/go-rest-boilerplate/internal/testenv"
	"github.com/snykk/go-rest-boilerplate/migrations"
)

const (
	// jwtSecret is a JWT secret of the 32 bytes the identity module requires at least.
	jwtSecret = "0123456789abcdef0123456789abcdef"
	// unusedDatabaseURL fills the DATABASE_URL the configuration requires; the pool comes from testenv.
	unusedDatabaseURL = "postgres://unused"
)

// content is what the seed leaves in the database.
type content struct {
	VerifiedUsers  int
	Editors        int
	Published      int
	Drafts         int
	Comments       int
	ReaderComments int // the reader's comments on the published article hello-world
}

type fixture struct {
	config   app.Config
	deps     app.Deps
	accounts Accounts
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	pool := testenv.Postgres(t, migrations.FS)
	config, err := app.LoadConfig(mapSource(map[string]string{
		"DATABASE_URL":         unusedDatabaseURL,
		"IDENTITY_JWT_SECRET":  jwtSecret,
		"IDENTITY_BCRYPT_COST": "10",
	}))
	if err != nil {
		t.Fatalf("LoadConfig = %v", err)
	}
	providers, err := telemetry.New(t.Context(), telemetry.Config{Service: "test"})
	if err != nil {
		t.Fatalf("telemetry.New = %v", err)
	}

	return &fixture{
		config: config,
		deps: app.Deps{
			Log:       slog.New(slog.DiscardHandler),
			Level:     new(slog.LevelVar),
			Pool:      pool,
			Telemetry: providers,
			Mail:      &mailer.Recorder{},
		},
		accounts: Accounts{
			EditorEmail:    "editor@example.com",
			EditorPassword: "editor-password",
			ReaderEmail:    "reader@example.com",
			ReaderPassword: "reader-password",
		},
	}
}

// content reads what the seed left, with SQL.
func (f *fixture) content(t *testing.T) content {
	t.Helper()
	var got content
	err := f.deps.Pool.QueryRow(t.Context(), `
		SELECT
			(SELECT count(*) FROM users WHERE verified_at IS NOT NULL),
			(SELECT count(*) FROM users WHERE 'editor' = ANY (roles)),
			(SELECT count(*) FROM articles WHERE status = 'published'),
			(SELECT count(*) FROM articles WHERE status = 'draft'),
			(SELECT count(*) FROM comments),
			(SELECT count(*)
				FROM comments c
				JOIN articles a ON a.id = c.article_id
				JOIN users u ON u.id = c.author_id
				WHERE a.slug = 'hello-world' AND a.status = 'published' AND u.email = $1)`,
		f.accounts.ReaderEmail,
	).Scan(&got.VerifiedUsers, &got.Editors, &got.Published, &got.Drafts, &got.Comments, &got.ReaderComments)
	if err != nil {
		t.Fatalf("count the seeded content = %v", err)
	}
	return got
}

// Seed provisions a verified editor and a verified reader, then, through the application's handler and signed
// in as those accounts, creates two published articles and a draft and has the reader comment twice on the
// first published article. A second run finds all of it in place, creates nothing and returns nil. A run after
// an interrupted one, which stopped at any of its steps, finishes what the first left: it publishes an article
// left a draft and adds what is missing, and the content is the same as after an uninterrupted run. A case's
// interrupt is the SQL that turns the content of a finished run into what an interrupted one leaves; a case
// without one has the second run follow the first at once.
func TestSeed(t *testing.T) {
	want := content{VerifiedUsers: 2, Editors: 1, Published: 2, Drafts: 1, Comments: 2, ReaderComments: 2}
	tCases := []struct {
		name      string
		interrupt string
	}{
		{name: "a second run creates nothing"},
		{
			name: "hello-world left a draft before it was published is published",
			interrupt: `
				DELETE FROM comments;
				DELETE FROM articles WHERE slug = 'second-post';
				UPDATE articles SET status = 'draft', published_at = NULL WHERE slug = 'hello-world'`,
		},
		{
			name: "second-post left a draft before it was published is published",
			interrupt: `
				DELETE FROM comments;
				UPDATE articles SET status = 'draft', published_at = NULL WHERE slug = 'second-post'`,
		},
		{
			name:      "a comment the interrupted run did not reach is added",
			interrupt: `DELETE FROM comments WHERE id = (SELECT id FROM comments ORDER BY created_at, id LIMIT 1)`,
		},
		{
			name:      "a run that stopped after the editor was created creates the rest",
			interrupt: `DELETE FROM comments; DELETE FROM articles; DELETE FROM users WHERE NOT 'editor' = ANY (roles)`,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if err := Seed(t.Context(), f.config, f.deps, f.accounts); err != nil {
				t.Fatalf("Seed = %v, want nil", err)
			}
			if got := f.content(t); got != want {
				t.Fatalf("content after the first run = %+v, want %+v", got, want)
			}
			if tc.interrupt != "" {
				if _, err := f.deps.Pool.Exec(t.Context(), tc.interrupt); err != nil {
					t.Fatalf("interrupt the run = %v", err)
				}
				if got := f.content(t); got == want {
					t.Fatalf("content after the interruption = %+v, want it to differ", got)
				}
			}

			if err := Seed(t.Context(), f.config, f.deps, f.accounts); err != nil {
				t.Fatalf("Seed again = %v, want nil", err)
			}

			if got := f.content(t); got != want {
				t.Errorf("content after the second run = %+v, want %+v", got, want)
			}
		})
	}
}
