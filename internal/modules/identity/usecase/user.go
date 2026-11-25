package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// UserView is a user as the use cases return it: plain data, so no caller outside the use cases holds the
// entity or changes it. It never carries the password hash.
type UserView struct {
	ID, Email, DisplayName string
	Roles                  []string
	Verified               bool
	CreatedAt              time.Time
}

// TokenPair is what a sign-in, a refresh and a password change hand to the client: a short-lived access token
// and the refresh token that replaces it.
type TokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// Settings are the lifetimes and limits the identity use cases work with.
type Settings struct {
	// AccessTTL is the lifetime of an access token. The Tokens implementation applies it, and the use cases
	// report the expiry Tokens.Issue returns, so no use case reads this field.
	AccessTTL time.Duration
	// RefreshTTL is the lifetime of a refresh token.
	RefreshTTL time.Duration
	// CodeTTL is the lifetime of an e-mail verification code.
	CodeTTL time.Duration
	// ResetTTL is the lifetime of a password reset token.
	ResetTTL time.Duration
	// ReuseGrace is how long after a refresh token was spent a client that did not see the answer may still
	// present it, without the token being taken as stolen.
	ReuseGrace time.Duration
	// Lockout is how long an account stays locked once LoginAttempts sign-ins have failed.
	Lockout time.Duration
	// CodeAttempts is how many wrong guesses a code takes: once this many were made, the code is refused as
	// domain.ErrTooManyAttempts, even for the right guess.
	CodeAttempts int
	// LoginAttempts is how many failed sign-ins lock an account.
	LoginAttempts int
	// ResendCooldown is how long after a verification code or a reset token was issued to a user no new one of
	// the same kind is issued to them: ResendVerification and ForgotPassword answer as usual, but store and mail
	// nothing.
	ResendCooldown time.Duration
}

// UserDeps are the ports and the settings a UserUsecase works with. Users, Verifications, Resets and Sessions
// serve the calls made outside a unit of work; inside one, the use case calls only the repositories
// UnitOfWork hands it.
type UserDeps struct {
	Users         UserRepository
	Verifications SecretRepository
	Resets        SecretRepository
	Sessions      SessionRepository
	UnitOfWork    UnitOfWork
	Hasher        PasswordHasher
	Tokens        Tokens
	Random        Randomness
	Mail          Mail
	Clock         Clock
	IDs           IDGenerator
	Settings      Settings
	// Log receives the security events: sign-ins that succeed or fail, lockouts, reused refresh tokens, and
	// passwords changed or reset.
	Log *slog.Logger
}

// UserUsecase runs the identity use cases. Every method that returns a UserView or a TokenPair returns the
// zero value together with a non-nil error. It is safe for concurrent use when its ports are.
type UserUsecase struct {
	users         UserRepository
	verifications SecretRepository
	resets        SecretRepository
	sessions      SessionRepository
	unitOfWork    UnitOfWork
	hasher        PasswordHasher
	tokens        Tokens
	random        Randomness
	mail          Mail
	clock         Clock
	ids           IDGenerator
	settings      Settings
	log           *slog.Logger
}

// NewUserUsecase returns a UserUsecase working with deps.
func NewUserUsecase(deps UserDeps) *UserUsecase {
	return &UserUsecase{
		users:         deps.Users,
		verifications: deps.Verifications,
		resets:        deps.Resets,
		sessions:      deps.Sessions,
		unitOfWork:    deps.UnitOfWork,
		hasher:        deps.Hasher,
		tokens:        deps.Tokens,
		random:        deps.Random,
		mail:          deps.Mail,
		clock:         deps.Clock,
		ids:           deps.IDs,
		settings:      deps.Settings,
		log:           deps.Log,
	}
}

func viewOf(user *domain.User) UserView {
	return UserView{
		ID:          user.ID(),
		Email:       user.Email().String(),
		DisplayName: user.DisplayName(),
		Roles:       user.Roles(),
		Verified:    user.Verified(),
		CreatedAt:   user.CreatedAt(),
	}
}

func hashOf(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func newSecret(userID, value string, now time.Time, ttl time.Duration) domain.Secret {
	return domain.Secret{UserID: userID, Hash: hashOf(value), ExpiresAt: now.Add(ttl), CreatedAt: now}
}

func (u *UserUsecase) newUser(
	email, password, displayName string, roles []string, now time.Time,
) (*domain.User, error) {
	address, err := domain.NewEmail(email)
	if err != nil {
		return nil, err
	}
	plain, err := domain.NewPassword(password)
	if err != nil {
		return nil, err
	}

	hash, err := u.hasher.Hash(plain.Reveal())
	if err != nil {
		return nil, err
	}
	return domain.NewUser(u.ids.NewID(), address, displayName, hash, roles, now)
}

// The security events the use cases log, as the event attribute of a "security event" record.
const (
	eventLoginSucceeded  = "login_succeeded"
	eventLoginFailed     = "login_failed"
	eventAccountLocked   = "account_locked"
	eventRefreshReused   = "refresh_reused"
	eventPasswordChanged = "password_changed"
	eventPasswordReset   = "password_reset"
)

// securityEvent logs event about the user with the given ID as a "security event" record at level, with the
// attributes event and user_id; a record about no known user, userID "", has no user_id. A record never holds
// an email or a submitted value.
func (u *UserUsecase) securityEvent(ctx context.Context, level slog.Level, event, userID string) {
	attrs := []slog.Attr{slog.String("event", event)}
	if userID != "" {
		attrs = append(attrs, slog.String("user_id", userID))
	}
	u.log.LogAttrs(ctx, level, "security event", attrs...)
}

// recordFailure counts a failed sign-in against the user, which locks the account when it is the last the
// settings allow, and returns the error the attempt answers: domain.ErrAccountLocked when the account was
// already locked when the failure was saved, as it is for an attempt that read the account before a concurrent
// one locked it, so every answer after the lock is the same; otherwise domain.ErrInvalidCredentials, also for
// the failure that takes the lock. Once the count is saved, it logs the failure as a security event, and the
// lock as another when this failure is the one that took it. It returns the store's error when the count
// cannot be saved.
func (u *UserUsecase) recordFailure(ctx context.Context, userID string, now time.Time) error {
	var wasLocked, locked bool
	_, err := u.users.Update(ctx, userID, func(user *domain.User) error {
		wasLocked = user.Locked(now)
		user.FailLogin(now, u.settings.LoginAttempts, u.settings.Lockout)
		locked = !wasLocked && user.Locked(now)
		return nil
	})
	if err != nil {
		return err
	}

	u.securityEvent(ctx, slog.LevelWarn, eventLoginFailed, userID)
	if locked {
		u.securityEvent(ctx, slog.LevelWarn, eventAccountLocked, userID)
	}
	if wasLocked {
		return domain.ErrAccountLocked
	}
	return domain.ErrInvalidCredentials
}

// lookup returns the user with the given email. It reports found false, and no error, for text that is no
// email as well as for an email no user has.
func (u *UserUsecase) lookup(ctx context.Context, email string) (user *domain.User, found bool, err error) {
	address, err := domain.NewEmail(email)
	if err != nil {
		return nil, false, nil
	}

	user, err = u.users.ByEmail(ctx, address)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return user, true, nil
}

// newSession builds a session that is not stored yet, and the refresh token it is for. The caller sets the
// session's user and family, or leaves them to the store, which takes them from the session it replaces.
func (u *UserUsecase) newSession(now time.Time) (session domain.Session, refreshToken string) {
	refreshToken = u.random.Token()
	session = domain.Session{
		ID:        u.ids.NewID(),
		TokenHash: hashOf(refreshToken),
		ExpiresAt: now.Add(u.settings.RefreshTTL),
		CreatedAt: now,
	}
	return session, refreshToken
}

// newFamily builds a session that starts a new family of user's sessions, and the refresh token it is for.
func (u *UserUsecase) newFamily(user *domain.User, now time.Time) (session domain.Session, refreshToken string) {
	session, refreshToken = u.newSession(now)
	session.UserID, session.Family = user.ID(), u.ids.NewID()
	return session, refreshToken
}

func (u *UserUsecase) pairOf(user *domain.User, refreshToken string, refreshExpiresAt time.Time) (TokenPair, error) {
	accessToken, accessExpiresAt, err := u.tokens.Issue(user.ID(), user.Roles())
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}
