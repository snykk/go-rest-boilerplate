package users_test

import (
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/snykk/go-rest-boilerplate/internal/test/mocks"
	"golang.org/x/crypto/bcrypt"
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
		// MinCost keeps Store's bcrypt hashing fast in tests (~1ms vs
		// ~80ms at the production cost of 12). Behaviour is identical.
		usecase: users.NewUsecase(repo, rc, users.Config{BcryptCost: bcrypt.MinCost}),
		repo:    repo,
		rc:      rc,
	}
}

// sampleUser returns a stable UserDomain used as canned repo output.
func sampleUser() domain.User {
	return domain.User{
		ID:        "11111111-1111-1111-1111-111111111111",
		Username:  "patrick",
		Email:     "patrick@example.com",
		Password:  "$2a$10$hashedpasswordhashedpasswordhash",
		Active:    true,
		RoleID:    2,
		CreatedAt: time.Now(),
	}
}
