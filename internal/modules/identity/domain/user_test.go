package domain_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

var baseTime = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)

const lockout = 15 * time.Minute

func emailFixture(t *testing.T) domain.Email {
	t.Helper()
	email, err := domain.NewEmail("ada@example.com")
	if err != nil {
		t.Fatalf("NewEmail = %v", err)
	}
	return email
}

// userFixture is an unverified user named Ada with the role reader, created at baseTime.
func userFixture(t *testing.T) *domain.User {
	t.Helper()
	user, err := domain.NewUser("user-1", emailFixture(t), "Ada", "hash-1", []string{"reader"}, baseTime)
	if err != nil {
		t.Fatalf("NewUser = %v", err)
	}
	return user
}

// failTimes returns count failure times, all at failedAt.
func failTimes(failedAt time.Time, count int) []time.Time {
	times := make([]time.Time, count)
	for i := range times {
		times[i] = failedAt
	}
	return times
}

// NewUser returns an unverified, unlocked user created at the given time, with the display name trimmed, and
// refuses a display name outside 1 to 50 characters or holding a control character.
func TestNewUser(t *testing.T) {
	email := emailFixture(t)
	bad := domain.ErrInvalidDisplayName
	fifty := strings.Repeat("é", 50)
	tCases := []struct {
		name        string
		displayName string
		want        string
		wantErr     error
	}{
		{name: "a name is kept", displayName: "Ada", want: "Ada"},
		{name: "a name is trimmed", displayName: "  Ada Lovelace \n", want: "Ada Lovelace"},
		{name: "50 multi-byte characters are accepted", displayName: fifty, want: fifty},
		{name: "an empty name is refused", displayName: "", wantErr: bad},
		{name: "a blank name is refused", displayName: " \t ", wantErr: bad},
		{name: "51 characters are refused", displayName: strings.Repeat("é", 51), wantErr: bad},
		{name: "a tab inside is refused", displayName: "Ada\tLovelace", wantErr: bad},
		{name: "a line break inside is refused", displayName: "Ada\nLovelace", wantErr: bad},
		{name: "a NUL character is refused", displayName: "Ada\x00", wantErr: bad},
		{name: "a DEL character is refused", displayName: "Ada\x7f", wantErr: bad},
		{name: "bytes that are not UTF-8 are refused", displayName: "Ada\xff", wantErr: bad},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.NewUser("user-1", email, tc.displayName, "hash-1", nil, baseTime)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("NewUser(%q) = %v, want %v", tc.displayName, err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if got != nil {
					t.Errorf("NewUser(%q) = %+v, want no user", tc.displayName, got.Stored())
				}
				return
			}
			if got.DisplayName() != tc.want {
				t.Errorf("NewUser(%q).DisplayName() = %q, want %q", tc.displayName, got.DisplayName(), tc.want)
			}
		})
	}
}

// Restore returns the user a store kept, field for field, without checking the rules again, so a user stored
// under an older rule still loads.
func TestUserRestore(t *testing.T) {
	email := emailFixture(t)
	oneHourLater := baseTime.Add(time.Hour)
	twoHoursLater := baseTime.Add(2 * time.Hour)
	tCases := []struct {
		name   string
		stored domain.UserStored
	}{
		{
			name: "a stored unverified user comes back as it was stored",
			stored: domain.UserStored{
				ID:           "user-1",
				Email:        email,
				DisplayName:  "Ada",
				PasswordHash: "hash-1",
				Roles:        []string{"reader"},
				CreatedAt:    baseTime,
				UpdatedAt:    oneHourLater,
			},
		},
		{
			name: "a stored verified and locked user comes back with its times and count",
			stored: domain.UserStored{
				ID:           "user-2",
				Email:        email,
				DisplayName:  "Grace",
				PasswordHash: "hash-2",
				Roles:        []string{"reader", "editor"},
				VerifiedAt:   &oneHourLater,
				FailedLogins: 5,
				LockedUntil:  &twoHoursLater,
				CreatedAt:    baseTime,
				UpdatedAt:    twoHoursLater,
			},
		},
		{
			name: "a user that breaks a newer rule still loads",
			stored: domain.UserStored{
				ID:           "user-3",
				Email:        email,
				DisplayName:  "",
				PasswordHash: "",
				CreatedAt:    baseTime,
				UpdatedAt:    baseTime,
			},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			got := domain.Restore(tc.stored)

			if stored := got.Stored(); !reflect.DeepEqual(stored, tc.stored) {
				t.Errorf("Restore = %+v, want %+v", stored, tc.stored)
			}
		})
	}
}

// User keeps its own copy of the roles it was given, and hands out copies, so no caller can change them.
func TestUserRoles(t *testing.T) {
	given := []string{"reader"}
	user, err := domain.NewUser("user-1", emailFixture(t), "Ada", "hash-1", given, baseTime)
	if err != nil {
		t.Fatalf("NewUser = %v", err)
	}

	given[0] = "admin"
	user.Roles()[0] = "admin"
	user.Stored().Roles[0] = "admin"

	if got := user.Roles(); !reflect.DeepEqual(got, []string{"reader"}) {
		t.Errorf("Roles = %v, want %v", got, []string{"reader"})
	}

	stored := domain.UserStored{ID: "user-2", Roles: []string{"reader"}}
	restored := domain.Restore(stored)
	stored.Roles[0] = "admin"

	if got := restored.Roles(); !reflect.DeepEqual(got, []string{"reader"}) {
		t.Errorf("Restore(...).Roles = %v, want %v", got, []string{"reader"})
	}
}

// Accessors return what the user was built with.
func TestUserAccessors(t *testing.T) {
	email := emailFixture(t)
	user := userFixture(t)

	if user.ID() != "user-1" || user.Email() != email || user.DisplayName() != "Ada" ||
		user.PasswordHash() != "hash-1" || !user.CreatedAt().Equal(baseTime) || !user.UpdatedAt().Equal(baseTime) ||
		user.FailedLogins() != 0 {
		t.Errorf("accessors = %q, %q, %q, %q, %v, %v, %d; want user-1, %q, Ada, hash-1, %v, %v, 0",
			user.ID(), user.Email(), user.DisplayName(), user.PasswordHash(), user.CreatedAt(), user.UpdatedAt(),
			user.FailedLogins(), email, baseTime, baseTime)
	}
	if verifiedAt, ok := user.VerifiedAt(); ok {
		t.Errorf("VerifiedAt = %v, true; want no time, false", verifiedAt)
	}
	if lockedUntil, ok := user.LockedUntil(); ok {
		t.Errorf("LockedUntil = %v, true; want no time, false", lockedUntil)
	}
}

// Verify marks the user verified at the given time, once: verifying a verified user changes nothing.
func TestUserVerify(t *testing.T) {
	oneHourLater := baseTime.Add(time.Hour)
	twoHoursLater := baseTime.Add(2 * time.Hour)
	tCases := []struct {
		name           string
		verifyAt       []time.Time
		wantVerified   bool
		wantVerifiedAt time.Time
		wantUpdatedAt  time.Time
	}{
		{name: "a new user is not verified", wantUpdatedAt: baseTime},
		{
			name:           "verifying sets the time and the update time",
			verifyAt:       []time.Time{oneHourLater},
			wantVerified:   true,
			wantVerifiedAt: oneHourLater,
			wantUpdatedAt:  oneHourLater,
		},
		{
			name:           "verifying again keeps the first time",
			verifyAt:       []time.Time{oneHourLater, twoHoursLater},
			wantVerified:   true,
			wantVerifiedAt: oneHourLater,
			wantUpdatedAt:  oneHourLater,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			user := userFixture(t)

			for _, verifyAt := range tc.verifyAt {
				user.Verify(verifyAt)
			}

			verifiedAt, ok := user.VerifiedAt()
			if user.Verified() != tc.wantVerified || ok != tc.wantVerified || !verifiedAt.Equal(tc.wantVerifiedAt) {
				t.Errorf("Verify left verified %t, verified at %v (%t); want %t, %v",
					user.Verified(), verifiedAt, ok, tc.wantVerified, tc.wantVerifiedAt)
			}
			if !user.UpdatedAt().Equal(tc.wantUpdatedAt) {
				t.Errorf("Verify left updated at %v, want %v", user.UpdatedAt(), tc.wantUpdatedAt)
			}
		})
	}
}

// Locked is true while the given time is before the end of the lockout, and false once it is reached.
func TestUserLocked(t *testing.T) {
	until := baseTime.Add(lockout)
	before := until.Add(-time.Nanosecond)
	tCases := []struct {
		name        string
		lockedUntil *time.Time
		at          time.Time
		want        bool
	}{
		{name: "a user never locked is open", at: baseTime, want: false},
		{name: "a user is locked before the end of the lockout", lockedUntil: &until, at: before, want: true},
		{name: "a user is open at the end of the lockout", lockedUntil: &until, at: until, want: false},
		{name: "a user is open after the lockout", lockedUntil: &until, at: until.Add(time.Hour), want: false},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			user := domain.Restore(domain.UserStored{ID: "user-1", LockedUntil: tc.lockedUntil})

			if got := user.Locked(tc.at); got != tc.want {
				t.Errorf("Locked(%v) = %t, want %t", tc.at, got, tc.want)
			}
		})
	}
}

// FailLogin counts a failed sign-in and locks the user, until the time plus the lockout, once the count
// reaches the maximum; once a lockout has ended, the next failure starts the count over.
func TestUserFailLogin(t *testing.T) {
	const maxAttempts = 5
	afterLockout := baseTime.Add(lockout + time.Minute)
	tCases := []struct {
		name         string
		failedAt     []time.Time
		at           time.Time
		wantFailures int
		wantLocked   bool
		wantUntil    time.Time
	}{
		{
			name:         "four failures with a maximum of five leave the user open",
			failedAt:     failTimes(baseTime, 4),
			at:           baseTime,
			wantFailures: 4,
		},
		{
			name:         "the fifth failure locks the user until the lockout ends",
			failedAt:     failTimes(baseTime, 5),
			at:           baseTime,
			wantFailures: 5,
			wantLocked:   true,
			wantUntil:    baseTime.Add(lockout),
		},
		{
			name:         "the user is locked just before the lockout ends",
			failedAt:     failTimes(baseTime, 5),
			at:           baseTime.Add(lockout - time.Nanosecond),
			wantFailures: 5,
			wantLocked:   true,
			wantUntil:    baseTime.Add(lockout),
		},
		{
			name:         "the user is open after the lockout",
			failedAt:     failTimes(baseTime, 5),
			at:           afterLockout,
			wantFailures: 5,
			wantUntil:    baseTime.Add(lockout),
		},
		{
			name:         "after a lockout ends, one wrong password does not lock again",
			failedAt:     append(failTimes(baseTime, 5), afterLockout),
			at:           afterLockout,
			wantFailures: 1,
		},
		{
			name:         "after a lockout ends, the fifth wrong password locks again",
			failedAt:     append(failTimes(baseTime, 5), failTimes(afterLockout, 5)...),
			at:           afterLockout,
			wantFailures: 5,
			wantLocked:   true,
			wantUntil:    afterLockout.Add(lockout),
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			user := userFixture(t)

			for _, failedAt := range tc.failedAt {
				user.FailLogin(failedAt, maxAttempts, lockout)
			}

			until, _ := user.LockedUntil()
			if user.FailedLogins() != tc.wantFailures || user.Locked(tc.at) != tc.wantLocked ||
				!until.Equal(tc.wantUntil) {
				t.Errorf("FailLogin left %d failures, locked at %v %t, locked until %v; want %d, %t, %v",
					user.FailedLogins(), tc.at, user.Locked(tc.at), until, tc.wantFailures, tc.wantLocked, tc.wantUntil)
			}
		})
	}
}

// SucceedLogin resets the count of failed sign-ins, so the next failures count from zero.
func TestUserSucceedLogin(t *testing.T) {
	user := userFixture(t)
	for range 4 {
		user.FailLogin(baseTime, 5, lockout)
	}

	user.SucceedLogin()

	if user.FailedLogins() != 0 {
		t.Fatalf("SucceedLogin left %d failures, want 0", user.FailedLogins())
	}
	for range 4 {
		user.FailLogin(baseTime, 5, lockout)
	}
	if user.Locked(baseTime) {
		t.Errorf("four failures after SucceedLogin locked the user, want it open")
	}
}

// SetPasswordHash replaces the password hash and moves the update time.
func TestUserSetPasswordHash(t *testing.T) {
	later := baseTime.Add(time.Hour)
	user := userFixture(t)

	user.SetPasswordHash("hash-2", later)

	if user.PasswordHash() != "hash-2" || !user.UpdatedAt().Equal(later) {
		t.Errorf("SetPasswordHash left hash %q, updated at %v; want hash-2, %v",
			user.PasswordHash(), user.UpdatedAt(), later)
	}
}

// ResetPassword replaces the password hash, moves the update time and ends any lockout: the count of failed
// sign-ins starts over and a locked user is open at once.
func TestUserResetPassword(t *testing.T) {
	later := baseTime.Add(time.Minute)
	tCases := []struct {
		name     string
		failures int
	}{
		{name: "a user with no failed sign-in gets the new hash", failures: 0},
		{name: "a user with failed sign-ins starts the count over", failures: 3},
		{name: "a locked user is open at once", failures: 5},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			user := userFixture(t)
			for range tc.failures {
				user.FailLogin(baseTime, 5, lockout)
			}

			user.ResetPassword("hash-2", later)

			_, lockedBefore := user.LockedUntil()
			if user.PasswordHash() != "hash-2" || !user.UpdatedAt().Equal(later) || user.FailedLogins() != 0 ||
				user.Locked(later) || lockedBefore {
				t.Errorf("ResetPassword left hash %q, updated at %v, %d failures, locked %t, a lockout on record %t; "+
					"want hash-2, %v, 0, false, false",
					user.PasswordHash(), user.UpdatedAt(), user.FailedLogins(), user.Locked(later), lockedBefore, later)
			}
		})
	}
}
