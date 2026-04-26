//go:build integration

package v1_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	V1Domains "github.com/snykk/go-rest-boilerplate/internal/business/domains/v1"
	postgresrepo "github.com/snykk/go-rest-boilerplate/internal/datasources/repositories/postgres/v1"
	"github.com/snykk/go-rest-boilerplate/internal/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixture builds a UserDomain with sensible defaults for tests.
// Caller overrides only what's relevant to its scenario.
func fixture(email string) *V1Domains.UserDomain {
	return &V1Domains.UserDomain{
		Username:  "user_" + email,
		Email:     email,
		Password:  "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
		RoleID:    2,
		CreatedAt: time.Now().UTC(),
	}
}

func TestRepo_StoreAndGetByEmail(t *testing.T) {
	db := testenv.StartPostgres(t)
	repo := postgresrepo.NewUserRepository(db)
	ctx := context.Background()

	stored, err := repo.Store(ctx, fixture("alice@example.com"))
	require.NoError(t, err)
	assert.NotEmpty(t, stored.ID, "INSERT … RETURNING must populate the id")
	assert.Equal(t, "alice@example.com", stored.Email)
	assert.False(t, stored.Active, "new users start inactive")

	got, err := repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: "alice@example.com"})
	require.NoError(t, err)
	assert.Equal(t, stored.ID, got.ID)
}

func TestRepo_StoreDuplicateEmail_ReturnsConflict(t *testing.T) {
	db := testenv.StartPostgres(t)
	repo := postgresrepo.NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.Store(ctx, fixture("dup@example.com"))
	require.NoError(t, err)

	// Same email, different username — partial unique index on email
	// must still trip on this.
	second := fixture("dup@example.com")
	second.Username = "another"
	_, err = repo.Store(ctx, second)
	require.Error(t, err)

	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr), "expected typed *apperror.DomainError, got %T", err)
	assert.Equal(t, apperror.ErrTypeConflict, domErr.Type)
}

func TestRepo_GetByEmail_NotFound(t *testing.T) {
	db := testenv.StartPostgres(t)
	repo := postgresrepo.NewUserRepository(db)

	_, err := repo.GetByEmail(context.Background(), &V1Domains.UserDomain{Email: "nobody@example.com"})
	require.Error(t, err)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeNotFound, domErr.Type)
}

func TestRepo_GetByID_RoundTrip(t *testing.T) {
	db := testenv.StartPostgres(t)
	repo := postgresrepo.NewUserRepository(db)
	ctx := context.Background()

	stored, err := repo.Store(ctx, fixture("byid@example.com"))
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, stored.ID)
	require.NoError(t, err)
	assert.Equal(t, stored.Email, got.Email)
}

func TestRepo_SoftDelete_HidesFromQueries(t *testing.T) {
	db := testenv.StartPostgres(t)
	repo := postgresrepo.NewUserRepository(db)
	ctx := context.Background()

	stored, err := repo.Store(ctx, fixture("gone@example.com"))
	require.NoError(t, err)

	require.NoError(t, repo.SoftDelete(ctx, stored.ID))

	// Default queries (GetByEmail / GetByID) must filter on
	// deleted_at IS NULL — the row exists in the table but should be
	// invisible to login / lookup paths.
	_, err = repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: "gone@example.com"})
	require.Error(t, err)
	_, err = repo.GetByID(ctx, stored.ID)
	require.Error(t, err)

	// Re-deleting an already-deleted row should report NotFound, not
	// silently succeed (which would mask bugs upstream).
	err = repo.SoftDelete(ctx, stored.ID)
	var domErr *apperror.DomainError
	require.True(t, errors.As(err, &domErr))
	assert.Equal(t, apperror.ErrTypeNotFound, domErr.Type)
}

func TestRepo_SoftDelete_AllowsReregistration(t *testing.T) {
	db := testenv.StartPostgres(t)
	repo := postgresrepo.NewUserRepository(db)
	ctx := context.Background()

	stored, err := repo.Store(ctx, fixture("recycle@example.com"))
	require.NoError(t, err)
	require.NoError(t, repo.SoftDelete(ctx, stored.ID))

	// The partial unique index on email is WHERE deleted_at IS NULL,
	// so the same email should be reusable after a soft delete.
	_, err = repo.Store(ctx, fixture("recycle@example.com"))
	require.NoError(t, err, "email should be reusable after soft delete")
}

func TestRepo_List_FiltersAndPagination(t *testing.T) {
	db := testenv.StartPostgres(t)
	repo := postgresrepo.NewUserRepository(db)
	ctx := context.Background()

	// Seed a mix of roles and active states.
	for i, email := range []string{"a@x.com", "b@x.com", "c@x.com"} {
		u := fixture(email)
		if i == 0 {
			u.RoleID = 1 // admin
		}
		_, err := repo.Store(ctx, u)
		require.NoError(t, err)
	}
	// Activate one user so ActiveOnly filter has something to hit.
	all, err := repo.List(ctx, V1Domains.ListFilter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, all, 3)

	require.NoError(t, repo.ChangeActiveUser(ctx, &V1Domains.UserDomain{ID: all[0].ID, Active: true}))

	t.Run("filter by role", func(t *testing.T) {
		got, err := repo.List(ctx, V1Domains.ListFilter{RoleID: 1}, 0, 10)
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})

	t.Run("filter active only", func(t *testing.T) {
		got, err := repo.List(ctx, V1Domains.ListFilter{ActiveOnly: true}, 0, 10)
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})

	t.Run("pagination", func(t *testing.T) {
		page1, err := repo.List(ctx, V1Domains.ListFilter{}, 0, 2)
		require.NoError(t, err)
		page2, err := repo.List(ctx, V1Domains.ListFilter{}, 2, 2)
		require.NoError(t, err)
		assert.Len(t, page1, 2)
		assert.Len(t, page2, 1)
		// IDs should not overlap between pages.
		assert.NotEqual(t, page1[0].ID, page2[0].ID)
	})
}
