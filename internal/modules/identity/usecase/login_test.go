package usecase_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

// Login signs a verified user in with a new session family and refuses the others, checking in the order
// locked, wrong password, unverified, with the same error for an unknown email and a wrong password. Once a
// concurrent attempt has locked the account, every attempt saved after it is refused as locked, with a wrong
// password or the right one, verified or not. A sign-in that succeeds, one refused as wrong credentials and a
// failure that locks the account are logged as security events, with the user's ID when the email is a user's,
// and with no email or password.
func TestLogin(t *testing.T) {
	tCases := []struct {
		name     string
		email    string
		password string
		mock     func(f *fixture)
		check    func(t *testing.T, saved *domain.User)
		want     usecase.TokenPair
		wantErr  error
		// wantEvents are the security events logged, in order; none when nil.
		wantEvents []loggedEvent
	}{
		{
			name:     "a verified user gets a pair, and the session is created",
			email:    userEmail,
			password: plainPassword,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), emailFixture(f.t)).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(passwordHash, plainPassword).Return(true)
				f.expectUpdate(f.store.users, userFixture(f.t))
				f.tokens.EXPECT().Issue(userID, []string{"reader"}).Return(accessToken, accessExpiry, nil)
				f.store.sessions.EXPECT().Create(gomock.Any(), domain.Session{
					ID:        "id-1",
					UserID:    userID,
					Family:    "id-2",
					TokenHash: sha(randomToken),
					ExpiresAt: baseTime.Add(settings().RefreshTTL),
					CreatedAt: baseTime,
				}).Return(nil)
			},
			want:       pairFixture(),
			wantEvents: []loggedEvent{{"INFO", "login_succeeded", userID}},
		},
		{
			name:     "the email is normalized before the lookup",
			email:    "  Ada@Example.COM ",
			password: plainPassword,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), emailFixture(f.t)).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(true)
				f.expectUpdate(f.store.users, userFixture(f.t))
				f.tokens.EXPECT().Issue(gomock.Any(), gomock.Any()).Return(accessToken, accessExpiry, nil)
				f.store.sessions.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			},
			want:       pairFixture(),
			wantEvents: []loggedEvent{{"INFO", "login_succeeded", userID}},
		},
		{
			name:     "failed sign-ins are cleared by a successful one",
			email:    userEmail,
			password: plainPassword,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(failedFixture(f.t, 3), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(true)
				f.expectUpdate(f.store.users, failedFixture(f.t, 3))
				f.tokens.EXPECT().Issue(gomock.Any(), gomock.Any()).Return(accessToken, accessExpiry, nil)
				f.store.sessions.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			},
			check: func(t *testing.T, saved *domain.User) {
				if got := saved.FailedLogins(); got != 0 {
					t.Errorf("Login saved FailedLogins = %d, want 0", got)
				}
			},
			want:       pairFixture(),
			wantEvents: []loggedEvent{{"INFO", "login_succeeded", userID}},
		},
		{
			name:     "an unknown email compares against the dummy hash",
			email:    userEmail,
			password: plainPassword,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
				f.hasher.EXPECT().CompareDummy(plainPassword).Times(1)
			},
			wantErr:    domain.ErrInvalidCredentials,
			wantEvents: []loggedEvent{{"WARN", "login_failed", ""}},
		},
		{
			name:     "a locked account is refused before the password is compared",
			email:    userEmail,
			password: plainPassword,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(lockedFixture(f.t), nil)
			},
			wantErr: domain.ErrAccountLocked,
		},
		{
			name:     "a wrong password counts as a failed sign-in",
			email:    userEmail,
			password: "wrong horse",
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(passwordHash, "wrong horse").Return(false)
				f.expectUpdate(f.store.users, userFixture(f.t))
			},
			check: func(t *testing.T, saved *domain.User) {
				if got := saved.FailedLogins(); got != 1 {
					t.Errorf("Login saved FailedLogins = %d, want 1", got)
				}
				if got := saved.Locked(baseTime); got {
					t.Errorf("Login saved Locked(baseTime) = %v, want false", got)
				}
			},
			wantErr:    domain.ErrInvalidCredentials,
			wantEvents: []loggedEvent{{"WARN", "login_failed", userID}},
		},
		{
			name:     "the fifth wrong password locks the account",
			email:    userEmail,
			password: "wrong horse",
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(failedFixture(f.t, 4), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(false)
				f.expectUpdate(f.store.users, failedFixture(f.t, 4))
			},
			check: func(t *testing.T, saved *domain.User) {
				until, _ := saved.LockedUntil()
				if want := baseTime.Add(settings().Lockout); !until.Equal(want) {
					t.Errorf("Login saved LockedUntil = %v, want %v", until, want)
				}
				if got := saved.Locked(baseTime); !got {
					t.Errorf("Login saved Locked(baseTime) = %v, want true", got)
				}
			},
			wantErr:    domain.ErrInvalidCredentials,
			wantEvents: []loggedEvent{{"WARN", "login_failed", userID}, {"WARN", "account_locked", userID}},
		},
		{
			name:     "an unverified user with the right password is ErrEmailNotVerified",
			email:    userEmail,
			password: plainPassword,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(unverifiedFixture(f.t), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(true)
				f.expectUpdate(f.store.users, unverifiedFixture(f.t))
			},
			wantErr: domain.ErrEmailNotVerified,
		},
		{
			name:     "an unverified user's right password on an account locked meanwhile is ErrAccountLocked",
			email:    userEmail,
			password: plainPassword,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(unverifiedFixture(f.t), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(true)
				f.expectUpdate(f.store.users, userWith(f.t, func(stored *domain.UserStored) {
					until := baseTime.Add(10 * time.Minute)
					stored.VerifiedAt, stored.FailedLogins, stored.LockedUntil = nil, 5, &until
				}))
			},
			wantErr: domain.ErrAccountLocked,
		},
		{
			name:     "a wrong password saved on an account locked meanwhile is ErrAccountLocked and logs no new lock",
			email:    userEmail,
			password: "wrong horse",
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(false)
				f.expectUpdate(f.store.users, lockedFixture(f.t))
			},
			check: func(t *testing.T, saved *domain.User) {
				if got := saved.FailedLogins(); got != 6 {
					t.Errorf("Login saved FailedLogins = %d, want 6", got)
				}
				if got := saved.Locked(baseTime); !got {
					t.Errorf("Login saved Locked(baseTime) = %v, want true", got)
				}
			},
			wantErr:    domain.ErrAccountLocked,
			wantEvents: []loggedEvent{{"WARN", "login_failed", userID}},
		},
		{
			name:     "an unverified user with a wrong password is ErrInvalidCredentials",
			email:    userEmail,
			password: "wrong horse",
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(unverifiedFixture(f.t), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(false)
				f.expectUpdate(f.store.users, unverifiedFixture(f.t))
			},
			wantErr:    domain.ErrInvalidCredentials,
			wantEvents: []loggedEvent{{"WARN", "login_failed", userID}},
		},
		{
			name:       "a password of the wrong length is refused as wrong before the store is called",
			email:      userEmail,
			password:   strings.Repeat("a", 73),
			mock:       func(*fixture) {},
			wantErr:    domain.ErrInvalidCredentials,
			wantEvents: []loggedEvent{{"WARN", "login_failed", ""}},
		},
		{
			name:       "an invalid email is refused as wrong before the store is called",
			email:      "ada",
			password:   plainPassword,
			mock:       func(*fixture) {},
			wantErr:    domain.ErrInvalidCredentials,
			wantEvents: []loggedEvent{{"WARN", "login_failed", ""}},
		},
		{
			name:     "a store that cannot be reached fails the call",
			email:    userEmail,
			password: plainPassword,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(nil, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:     "a failed sign-in that cannot be saved fails the call",
			email:    userEmail,
			password: "wrong horse",
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(false)
				f.store.users.EXPECT().Update(gomock.Any(), userID, gomock.Any()).Return(nil, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:     "a successful sign-in that cannot be saved fails the call",
			email:    userEmail,
			password: plainPassword,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(true)
				f.store.users.EXPECT().Update(gomock.Any(), userID, gomock.Any()).Return(nil, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:     "a lock taken by a concurrent attempt is honoured on success",
			email:    userEmail,
			password: plainPassword,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(true)
				f.expectUpdate(f.store.users, lockedFixture(f.t))
			},
			wantErr: domain.ErrAccountLocked,
		},
		{
			name:     "an access token that cannot be signed stores no session",
			email:    userEmail,
			password: plainPassword,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(true)
				f.expectUpdate(f.store.users, userFixture(f.t))
				f.tokens.EXPECT().Issue(gomock.Any(), gomock.Any()).Return("", time.Time{}, errSigning)
			},
			wantErr: errSigning,
		},
		{
			name:     "a session that cannot be stored fails the call with no pair",
			email:    userEmail,
			password: plainPassword,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(true)
				f.expectUpdate(f.store.users, userFixture(f.t))
				f.tokens.EXPECT().Issue(gomock.Any(), gomock.Any()).Return(accessToken, accessExpiry, nil)
				f.store.sessions.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errStoreDown)
			},
			wantErr: errStoreDown,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			got, err := f.useCase.Login(t.Context(), tc.email, tc.password)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Login = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Login = %+v, want %+v", got, tc.want)
			}
			if got := f.securityEvents(t); !reflect.DeepEqual(got, tc.wantEvents) {
				t.Errorf("Login logged %+v, want %+v", got, tc.wantEvents)
			}
			if tc.check != nil {
				if f.updated == nil {
					t.Fatalf("Login saved user = nil, want a user")
				}
				tc.check(t, f.updated)
			}
		})
	}
}
