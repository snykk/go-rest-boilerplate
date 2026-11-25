// Package domain holds the identity entities and the rules every user, session and secret keeps, whatever
// stores or serves them.
package domain

import (
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const maxDisplayNameLen = 50

// User is an account that signs in with an e-mail address and a password. It starts unverified and unlocked.
// Its state changes only through NewUser and its methods, so a user in memory always keeps their rules.
type User struct {
	id           string
	email        Email
	displayName  string
	passwordHash string
	roles        []string
	verifiedAt   time.Time
	failedLogins int
	lockedUntil  time.Time
	createdAt    time.Time
	updatedAt    time.Time
}

// UserStored is the state of a user as a store keeps it, for Restore and for a store saving a user.
type UserStored struct {
	ID           string
	Email        Email
	DisplayName  string
	PasswordHash string
	Roles        []string
	// VerifiedAt is nil for a user whose e-mail is not verified.
	VerifiedAt   *time.Time
	FailedLogins int
	// LockedUntil is nil for a user who was never locked.
	LockedUntil *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewUser returns an unverified, unlocked user created at now, with the display name trimmed and roles
// copied. It returns ErrInvalidDisplayName unless the display name is 1 to 50 valid UTF-8 characters with no
// control character.
func NewUser(id string, email Email, displayName, hash string, roles []string, now time.Time) (*User, error) {
	displayName = strings.TrimSpace(displayName)
	if !validDisplayName(displayName) {
		return nil, ErrInvalidDisplayName
	}
	return &User{
		id:           id,
		email:        email,
		displayName:  displayName,
		passwordHash: hash,
		roles:        slices.Clone(roles),
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// Restore returns the user a store kept as stored. It is for gateways that load stored users, not a way to
// create one: it does not check the rules again, so a user stored under an older rule still loads.
func Restore(stored UserStored) *User {
	user := &User{
		id:           stored.ID,
		email:        stored.Email,
		displayName:  stored.DisplayName,
		passwordHash: stored.PasswordHash,
		roles:        slices.Clone(stored.Roles),
		failedLogins: stored.FailedLogins,
		createdAt:    stored.CreatedAt,
		updatedAt:    stored.UpdatedAt,
	}
	if stored.VerifiedAt != nil {
		user.verifiedAt = *stored.VerifiedAt
	}
	if stored.LockedUntil != nil {
		user.lockedUntil = *stored.LockedUntil
	}
	return user
}

// Stored returns the user's state as a store keeps it, for a gateway that saves the user. The roles are a
// copy.
func (u *User) Stored() UserStored {
	stored := UserStored{
		ID:           u.id,
		Email:        u.email,
		DisplayName:  u.displayName,
		PasswordHash: u.passwordHash,
		Roles:        slices.Clone(u.roles),
		FailedLogins: u.failedLogins,
		CreatedAt:    u.createdAt,
		UpdatedAt:    u.updatedAt,
	}
	if verifiedAt, ok := u.VerifiedAt(); ok {
		stored.VerifiedAt = &verifiedAt
	}
	if lockedUntil, ok := u.LockedUntil(); ok {
		stored.LockedUntil = &lockedUntil
	}
	return stored
}

// ID returns the user's identifier.
func (u *User) ID() string { return u.id }

// Email returns the user's normalized address.
func (u *User) Email() Email { return u.email }

// DisplayName returns the name shown for the user.
func (u *User) DisplayName() string { return u.displayName }

// PasswordHash returns the hash of the user's password.
func (u *User) PasswordHash() string { return u.passwordHash }

// Roles returns a copy of the user's roles.
func (u *User) Roles() []string { return slices.Clone(u.roles) }

// VerifiedAt returns when the user's e-mail was verified, and false for an unverified user.
func (u *User) VerifiedAt() (time.Time, bool) { return u.verifiedAt, u.Verified() }

// FailedLogins returns the number of failed sign-ins since the last successful one.
func (u *User) FailedLogins() int { return u.failedLogins }

// LockedUntil returns the end of the user's latest lockout, and false for a user with no lockout on record.
// The time may be in the past; Locked tells whether a lockout is on.
func (u *User) LockedUntil() (time.Time, bool) {
	return u.lockedUntil, !u.lockedUntil.IsZero()
}

// CreatedAt returns when the user was created.
func (u *User) CreatedAt() time.Time { return u.createdAt }

// UpdatedAt returns when the user last changed.
func (u *User) UpdatedAt() time.Time { return u.updatedAt }

// Verify marks the user's e-mail verified at now. A user who is already verified is left as it is.
func (u *User) Verify(now time.Time) {
	if u.Verified() {
		return
	}
	u.verifiedAt, u.updatedAt = now, now
}

// Verified reports whether the user's e-mail is verified.
func (u *User) Verified() bool {
	return !u.verifiedAt.IsZero()
}

// Locked reports whether now is before the end of the user's lockout.
func (u *User) Locked(now time.Time) bool {
	return now.Before(u.lockedUntil)
}

// FailLogin records a failed sign-in at now. Once the count of failures reaches maxAttempts the user is
// locked until now plus lockout. A lockout that has ended by now clears the lock and starts the count over,
// so the failure that follows it counts as the first.
func (u *User) FailLogin(now time.Time, maxAttempts int, lockout time.Duration) {
	if !u.lockedUntil.IsZero() && !u.Locked(now) {
		u.failedLogins, u.lockedUntil = 0, time.Time{}
	}

	u.failedLogins++
	u.updatedAt = now
	if u.failedLogins >= maxAttempts {
		u.lockedUntil = now.Add(lockout)
	}
}

// SucceedLogin resets the count of failed sign-ins.
func (u *User) SucceedLogin() {
	u.failedLogins = 0
}

// SetPasswordHash replaces the hash of the user's password at now.
func (u *User) SetPasswordHash(hash string, now time.Time) {
	u.passwordHash, u.updatedAt = hash, now
}

// ResetPassword replaces the hash of the user's password at now and ends any lockout: the count of failed
// sign-ins starts over and the lock is cleared. A reset proves that the user holds their mailbox, so failures
// someone else caused no longer keep them out.
func (u *User) ResetPassword(hash string, now time.Time) {
	u.SetPasswordHash(hash, now)
	u.failedLogins, u.lockedUntil = 0, time.Time{}
}

func validDisplayName(name string) bool {
	length := utf8.RuneCountInString(name)
	return utf8.ValidString(name) && length >= 1 && length <= maxDisplayNameLen &&
		!strings.ContainsFunc(name, unicode.IsControl)
}
