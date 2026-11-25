//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
	"github.com/snykk/go-rest-boilerplate/internal/testenv"
	"github.com/snykk/go-rest-boilerplate/migrations"
)

var baseTime = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC) // whole seconds: Postgres keeps microseconds

// later is when the sessions and secrets of the fixtures expire, unless a test says otherwise.
var later = baseTime.Add(time.Hour)

// grace is the reuse grace the tests rotate with.
const grace = 10 * time.Second

// newPool is a pool on a database of the test's own, migrated with the real migrations.
func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return testenv.Postgres(t, migrations.FS)
}

// newUser is an unverified reader with the given email, created at baseTime and not stored.
func newUser(t *testing.T, email string) *domain.User {
	t.Helper()
	address, err := domain.NewEmail(email)
	if err != nil {
		t.Fatalf("NewEmail(%q) = %v", email, err)
	}
	user, err := domain.NewUser(id.New(), address, "Ada", "bcrypt-hash", []string{"reader"}, baseTime)
	if err != nil {
		t.Fatalf("NewUser = %v", err)
	}
	return user
}

// storeUser stores a new user with the given email in pool, failing the test when it cannot.
func storeUser(t *testing.T, pool *pgxpool.Pool, email string) *domain.User {
	t.Helper()
	user := newUser(t, email)
	if err := postgres.NewUserRepo(pool).Create(t.Context(), user); err != nil {
		t.Fatalf("Create = %v", err)
	}
	return user
}

// insertUserRow stores a user with the given email exactly as written, as a store written under an older rule
// may hold it, and returns the user's ID.
func insertUserRow(t *testing.T, pool *pgxpool.Pool, email string) string {
	t.Helper()
	userID := id.New()
	_, err := pool.Exec(t.Context(),
		`INSERT INTO users (id, email, display_name, password_hash, roles, created_at, updated_at)
		 VALUES ($1, $2, 'Old', 'bcrypt-hash', '{reader}', $3, $3)`,
		userID, email, baseTime)
	if err != nil {
		t.Fatalf("insert user row = %v", err)
	}
	return userID
}

// sessionOf is a session of the user in family for the token hash, created at baseTime and expiring at expiresAt.
func sessionOf(userID, family, tokenHash string, expiresAt time.Time) domain.Session {
	return domain.Session{
		ID:        id.New(),
		UserID:    userID,
		Family:    family,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: baseTime,
	}
}

// successor is the session a rotation stores for the token hash: its user and family are left to Rotate.
func successor(tokenHash string) domain.Session {
	return domain.Session{
		ID:        id.New(),
		TokenHash: tokenHash,
		ExpiresAt: later,
		CreatedAt: baseTime,
	}
}

// storeSession stores session in pool, failing the test when it cannot.
func storeSession(t *testing.T, pool *pgxpool.Pool, session domain.Session) {
	t.Helper()
	if err := postgres.NewSessionRepo(pool).Create(t.Context(), session); err != nil {
		t.Fatalf("Create session = %v", err)
	}
}

// liveTokens returns the token hashes of the sessions in pool that are live at now, in order.
func liveTokens(t *testing.T, pool *pgxpool.Pool, now time.Time) []string {
	t.Helper()
	rows, err := pool.Query(t.Context(),
		`SELECT token_hash FROM sessions
		  WHERE spent_at IS NULL AND revoked_at IS NULL AND expires_at > $1
		  ORDER BY token_hash`, now)
	if err != nil {
		t.Fatalf("select live sessions = %v", err)
	}
	tokens, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("collect live sessions = %v", err)
	}
	return tokens
}

// ownerOf returns the user and the family of the session with the token hash.
func ownerOf(t *testing.T, pool *pgxpool.Pool, tokenHash string) (userID, family string) {
	t.Helper()
	err := pool.QueryRow(t.Context(), `SELECT user_id, family FROM sessions WHERE token_hash = $1`, tokenHash).
		Scan(&userID, &family)
	if err != nil {
		t.Fatalf("select session %q = %v", tokenHash, err)
	}
	return userID, family
}

// waitForLockWaiter returns once a query on pool's database waits for a lock, and fails the test when none does
// within five seconds.
func waitForLockWaiter(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var waiting int
		err := pool.QueryRow(t.Context(),
			`SELECT count(*) FROM pg_stat_activity
			  WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&waiting)
		if err != nil {
			t.Fatalf("select lock waiters = %v", err)
		}
		if waiting > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no query waited for a lock within five seconds")
}

// sessionCall is one call to a SessionRepo, for a race between two of them.
type sessionCall func(ctx context.Context, sessions *postgres.SessionRepo) error

// holdThenWait runs holder on a SessionRepo over a transaction it keeps open, starts waiter on a SessionRepo
// over pool, commits the transaction once waiter waits for a lock, and returns what each call returned.
func holdThenWait(t *testing.T, pool *pgxpool.Pool, holder, waiter sessionCall) (holderErr, waiterErr error) {
	t.Helper()
	transaction, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatalf("Begin = %v", err)
	}
	defer func() { _ = transaction.Rollback(context.Background()) }()

	holderErr = holder(t.Context(), postgres.NewSessionRepo(transaction))
	waited := make(chan error, 1)
	go func() { waited <- waiter(t.Context(), postgres.NewSessionRepo(pool)) }()
	waitForLockWaiter(t, pool)
	if err := transaction.Commit(t.Context()); err != nil {
		t.Fatalf("Commit = %v", err)
	}

	return holderErr, <-waited
}

// secretOf is a secret of the user with the hash, created at baseTime and expiring at later, with no wrong
// guess made.
func secretOf(userID, hash string) domain.Secret {
	return domain.Secret{UserID: userID, Hash: hash, ExpiresAt: later, CreatedAt: baseTime}
}

// put stores secret with repo, replacing any secret the user held, and fails the test when it cannot.
func put(t *testing.T, repo *postgres.SecretRepo, secret domain.Secret) {
	t.Helper()
	if stored, err := repo.Put(t.Context(), secret, 0); !stored || err != nil {
		t.Fatalf("Put = %t, %v; want true, nil", stored, err)
	}
}
