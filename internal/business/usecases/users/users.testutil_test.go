package users_test

import (
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/snykk/go-rest-boilerplate/internal/test/mocks"
)

// fixture is the per-test wiring. Each sub-test calls newFixture()
// to get a clean set of mocks — there's no shared mutable state
// between tests.
type fixture struct {
	usecase users.Usecase
	repo    *mocks.UserRepository
	rc      *mocks.RistrettoCache
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	repo := mocks.NewUserRepository(t)
	rc := mocks.NewRistrettoCache(t)
	return &fixture{
		usecase: users.NewUsecase(repo, rc),
		repo:    repo,
		rc:      rc,
	}
}

// sampleUser returns a stable UserDomain used as canned repo output.
func sampleUser() entities.UserDomain {
	return entities.UserDomain{
		ID:        "11111111-1111-1111-1111-111111111111",
		Username:  "patrick",
		Email:     "patrick@example.com",
		Password:  "$2a$10$hashedpasswordhashedpasswordhash",
		Active:    true,
		RoleID:    2,
		CreatedAt: time.Now(),
	}
}
