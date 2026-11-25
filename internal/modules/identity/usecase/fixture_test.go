package usecase_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

var baseTime = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)

const (
	userID           = "user-1"
	userEmail        = "ada@example.com"
	plainPassword    = "correct horse"
	passwordHash     = "hash-of-correct-horse"
	newPlainPassword = "battery staple"
	newPasswordHash  = "hash-of-battery-staple"
	verifyCode       = "123456"
	randomToken      = "random-token"
	oldRefreshToken  = "old-refresh-token"
	accessToken      = "access-token-1"
)

var (
	// errStoreDown is a store failing on its own: a connection it cannot make.
	errStoreDown = errors.New("dial tcp: connection refused")
	// errHashing is a password that cannot be hashed.
	errHashing = errors.New("bcrypt: out of memory")
	// errSigning is a token that cannot be signed.
	errSigning = errors.New("sign: key unavailable")
)

// accessExpiry is when the access token the fixture's Tokens issue expires.
var accessExpiry = baseTime.Add(15 * time.Minute)

func settings() usecase.Settings {
	return usecase.Settings{
		AccessTTL:      15 * time.Minute,
		RefreshTTL:     720 * time.Hour,
		CodeTTL:        10 * time.Minute,
		ResetTTL:       30 * time.Minute,
		ReuseGrace:     10 * time.Second,
		Lockout:        15 * time.Minute,
		CodeAttempts:   5,
		LoginAttempts:  5,
		ResendCooldown: time.Minute,
	}
}

// sha is the hex SHA-256 of value, the form in which the use cases store a code or a token.
func sha(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// fixedClock always answers baseTime.
type fixedClock struct{}

func (fixedClock) Now() time.Time { return baseTime }

// sequenceIDs answers "id-1", "id-2" and so on.
type sequenceIDs struct{ issued int }

func (s *sequenceIDs) NewID() string {
	s.issued++
	return fmt.Sprintf("id-%d", s.issued)
}

// fixedRandom always answers verifyCode as the code and randomToken as the token.
type fixedRandom struct{}

func (fixedRandom) Code() string  { return verifyCode }
func (fixedRandom) Token() string { return randomToken }

// mockRepositories is one set of mocked repositories.
type mockRepositories struct {
	users         *MockUserRepository
	verifications *MockSecretRepository
	resets        *MockSecretRepository
	sessions      *MockSessionRepository
}

func newMockRepositories(controller *gomock.Controller) *mockRepositories {
	return &mockRepositories{
		users:         NewMockUserRepository(controller),
		verifications: NewMockSecretRepository(controller),
		resets:        NewMockSecretRepository(controller),
		sessions:      NewMockSessionRepository(controller),
	}
}

func (m *mockRepositories) repositories() usecase.Repositories {
	return usecase.Repositories{
		Users:         m.users,
		Verifications: m.verifications,
		Resets:        m.resets,
		Sessions:      m.sessions,
	}
}

// fixture is a UserUsecase over mocked ports, fresh for every case. A call a case does not expect fails it.
// store is what the use case calls outside a unit of work and tx is what its unit of work hands out, two
// separate sets of mocks, so a write made on the wrong one fails the case.
type fixture struct {
	t          *testing.T
	store      *mockRepositories
	tx         *mockRepositories
	unitOfWork *MockUnitOfWork
	hasher     *MockPasswordHasher
	tokens     *MockTokens
	mail       *MockMail
	useCase    *usecase.UserUsecase
	committed  bool
	// updated is the user as the last expected Update saved it.
	updated *domain.User
	// logs holds the records the use case logs, in JSON.
	logs *bytes.Buffer
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	controller := gomock.NewController(t)
	f := &fixture{
		t:          t,
		store:      newMockRepositories(controller),
		tx:         newMockRepositories(controller),
		unitOfWork: NewMockUnitOfWork(controller),
		hasher:     NewMockPasswordHasher(controller),
		tokens:     NewMockTokens(controller),
		mail:       NewMockMail(controller),
		logs:       &bytes.Buffer{},
	}
	f.useCase = usecase.NewUserUsecase(usecase.UserDeps{
		Users:         f.store.users,
		Verifications: f.store.verifications,
		Resets:        f.store.resets,
		Sessions:      f.store.sessions,
		UnitOfWork:    f.unitOfWork,
		Hasher:        f.hasher,
		Tokens:        f.tokens,
		Random:        fixedRandom{},
		Mail:          f.mail,
		Clock:         fixedClock{},
		IDs:           &sequenceIDs{},
		Settings:      settings(),
		Log:           slog.New(slog.NewJSONHandler(f.logs, nil)),
	})
	return f
}

// loggedEvent is a security event as the use case logs it: its level, its event and the ID of the user it is
// about, "" when it names none.
type loggedEvent struct {
	level, event, userID string
}

// securityEvents returns the security events the use case logged, in order, or nil when it logged none. It
// fails t when the logs carry the fixture's email, in any case, one of its passwords or one of its tokens.
func (f *fixture) securityEvents(t *testing.T) []loggedEvent {
	t.Helper()
	logs := strings.ToLower(f.logs.String())
	for _, secret := range []string{userEmail, plainPassword, newPlainPassword, "wrong horse", randomToken, oldRefreshToken} {
		if strings.Contains(logs, secret) {
			t.Errorf("logs = %s, want %q kept out", f.logs, secret)
		}
	}

	var events []loggedEvent
	for line := range strings.Lines(f.logs.String()) {
		var record struct {
			Level  string `json:"level"`
			Msg    string `json:"msg"`
			Event  string `json:"event"`
			UserID string `json:"user_id"`
		}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("log line %q is not JSON: %v", line, err)
		}
		if record.Msg == "security event" {
			events = append(events, loggedEvent{level: record.Level, event: record.Event, userID: record.UserID})
		}
	}
	return events
}

// expectUnitOfWork makes the unit of work call its function with the tx repositories and return its error,
// and mark the fixture committed when that error is nil. It returns the expectation, to order other calls
// around it.
func (f *fixture) expectUnitOfWork() *gomock.Call {
	return f.unitOfWork.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, work func(usecase.Repositories) error) error {
			if err := work(f.tx.repositories()); err != nil {
				return err
			}
			f.committed = true
			return nil
		})
}

// expectUpdate makes the repository update the stored user as Update does: apply the change, then return
// the result and keep it as f.updated, or the change's error with nothing saved.
func (f *fixture) expectUpdate(users *MockUserRepository, stored *domain.User) *gomock.Call {
	return users.EXPECT().Update(gomock.Any(), stored.ID(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, change func(*domain.User) error) (*domain.User, error) {
			if err := change(stored); err != nil {
				return nil, err
			}
			f.updated = stored
			return stored, nil
		})
}

// expectUse makes the repository use the stored secret as Use does: run the check on a copy of it and
// return the check's error.
func (f *fixture) expectUse(secrets *MockSecretRepository, stored domain.Secret) *gomock.Call {
	return secrets.EXPECT().Use(gomock.Any(), stored.UserID, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, check func(*domain.Secret) error) error {
			secret := stored
			return check(&secret)
		})
}

// expectMailAfterCommit expects the verification mail, and fails the case when it is sent before the unit
// of work committed.
func (f *fixture) expectMailAfterCommit() *gomock.Call {
	return f.mail.EXPECT().Verification(gomock.Any(), emailFixture(f.t), verifyCode).Do(
		func(context.Context, domain.Email, string) {
			if !f.committed {
				f.t.Errorf("Verification called with the unit of work committed = %v, want true", f.committed)
			}
		})
}

func emailFixture(t *testing.T) domain.Email {
	t.Helper()
	email, err := domain.NewEmail(userEmail)
	if err != nil {
		t.Fatalf("NewEmail(%q) = %v, want nil", userEmail, err)
	}
	return email
}

// userWith is the stored user Ada, verified at baseTime with the role reader and the hash passwordHash, with one
// change made to its stored state.
func userWith(t *testing.T, change func(stored *domain.UserStored)) *domain.User {
	t.Helper()
	verified := baseTime
	stored := domain.UserStored{
		ID:           userID,
		Email:        emailFixture(t),
		DisplayName:  "Ada",
		PasswordHash: passwordHash,
		Roles:        []string{"reader"},
		VerifiedAt:   &verified,
		CreatedAt:    baseTime,
		UpdatedAt:    baseTime,
	}
	change(&stored)
	return domain.Restore(stored)
}

// userFixture is the verified user Ada as the store returns her.
func userFixture(t *testing.T) *domain.User {
	t.Helper()
	return userWith(t, func(*domain.UserStored) {})
}

// unverifiedFixture is Ada before she verified her email.
func unverifiedFixture(t *testing.T) *domain.User {
	t.Helper()
	return userWith(t, func(stored *domain.UserStored) { stored.VerifiedAt = nil })
}

// failedFixture is Ada after the given number of failed sign-ins, not locked yet.
func failedFixture(t *testing.T, count int) *domain.User {
	t.Helper()
	return userWith(t, func(stored *domain.UserStored) { stored.FailedLogins = count })
}

// lockedFixture is Ada locked out until ten minutes after baseTime, after five failed sign-ins.
func lockedFixture(t *testing.T) *domain.User {
	t.Helper()
	return userWith(t, func(stored *domain.UserStored) {
		until := baseTime.Add(10 * time.Minute)
		stored.FailedLogins, stored.LockedUntil = 5, &until
	})
}

// viewFixture is the UserView of userFixture.
func viewFixture() usecase.UserView {
	return usecase.UserView{
		ID:          userID,
		Email:       userEmail,
		DisplayName: "Ada",
		Roles:       []string{"reader"},
		Verified:    true,
		CreatedAt:   baseTime,
	}
}

// viewWith is viewFixture with one change made to it.
func viewWith(change func(view *usecase.UserView)) usecase.UserView {
	view := viewFixture()
	change(&view)
	return view
}

// pairFixture is the pair the use cases return for the tokens of this fixture.
func pairFixture() usecase.TokenPair {
	return usecase.TokenPair{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiry,
		RefreshToken:     randomToken,
		RefreshExpiresAt: baseTime.Add(settings().RefreshTTL),
	}
}

// codeFor is the secret Register and ResendVerification store for the user with the given ID: the hash of
// verifyCode, created at baseTime and valid for the code's lifetime.
func codeFor(id string) domain.Secret {
	return domain.Secret{
		UserID:    id,
		Hash:      sha(verifyCode),
		ExpiresAt: baseTime.Add(settings().CodeTTL),
		CreatedAt: baseTime,
	}
}

// resetFor is the secret ForgotPassword stores for the user with the given ID: the hash of randomToken,
// created at baseTime and valid for the token's lifetime.
func resetFor(id string) domain.Secret {
	return domain.Secret{
		UserID:    id,
		Hash:      sha(randomToken),
		ExpiresAt: baseTime.Add(settings().ResetTTL),
		CreatedAt: baseTime,
	}
}

// secretWith is secret with one change made to it.
func secretWith(secret domain.Secret, change func(secret *domain.Secret)) domain.Secret {
	change(&secret)
	return secret
}

// registeredID is the ID the fixture's generator gives the first thing a use case creates.
const registeredID = "id-1"

// registeredFixture is the unverified user Register builds from validInput.
func registeredFixture(t *testing.T) *domain.User {
	t.Helper()
	return userWith(t, func(stored *domain.UserStored) {
		stored.ID = registeredID
		stored.VerifiedAt = nil
	})
}
