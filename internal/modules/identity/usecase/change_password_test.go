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

// ChangePassword replaces the password of a user who knows the current one, revokes every session and starts
// a new one in one unit of work, and returns the pair of that session. A wrong current password counts as a
// failed attempt and changes nothing else, and a locked account is refused, also when the lock is taken before
// a wrong current password is saved, before a right one leads to hashing the new password, or between that
// hash and the change. A committed change, a wrong current password and a failure that locks the account are
// logged as security events with the user's ID.
func TestChangePassword(t *testing.T) {
	validInput := usecase.ChangePasswordInput{
		UserID:  userID,
		Current: plainPassword,
		New:     newPlainPassword,
	}
	inputWith := func(change func(input *usecase.ChangePasswordInput)) usecase.ChangePasswordInput {
		input := validInput
		change(&input)
		return input
	}
	expectCurrentRight := func(f *fixture) {
		f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(userFixture(f.t), nil)
		f.hasher.EXPECT().Compare(passwordHash, plainPassword).Return(true)
		f.expectUpdate(f.store.users, userFixture(f.t))
	}
	tCases := []struct {
		name    string
		input   usecase.ChangePasswordInput
		mock    func(f *fixture)
		check   func(t *testing.T, saved *domain.User)
		want    usecase.TokenPair
		wantErr error
		// wantEvents are the security events logged, in order; none when nil.
		wantEvents []loggedEvent
	}{
		{
			name:  "the new hash, the revoked sessions and the new session are written in one unit of work",
			input: validInput,
			mock: func(f *fixture) {
				expectCurrentRight(f)
				f.hasher.EXPECT().Hash(newPlainPassword).Return(newPasswordHash, nil)
				f.tokens.EXPECT().Issue(userID, []string{"reader"}).Return(accessToken, accessExpiry, nil)
				work := f.expectUnitOfWork()
				update := f.expectUpdate(f.tx.users, userFixture(f.t))
				revoke := f.tx.sessions.EXPECT().RevokeAll(gomock.Any(), userID, baseTime).Return(nil)
				create := f.tx.sessions.EXPECT().Create(gomock.Any(), domain.Session{
					ID:        "id-1",
					UserID:    userID,
					Family:    "id-2",
					TokenHash: sha(randomToken),
					ExpiresAt: baseTime.Add(settings().RefreshTTL),
					CreatedAt: baseTime,
				}).Return(nil)
				gomock.InOrder(work, update, revoke, create)
			},
			check: func(t *testing.T, saved *domain.User) {
				if got := saved.PasswordHash(); got != newPasswordHash {
					t.Errorf("ChangePassword saved PasswordHash = %q, want %q", got, newPasswordHash)
				}
			},
			want:       pairFixture(),
			wantEvents: []loggedEvent{{"INFO", "password_changed", userID}},
		},
		{
			name:  "a wrong current password counts a failed attempt and changes nothing else",
			input: inputWith(func(input *usecase.ChangePasswordInput) { input.Current = "wrong horse" }),
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(passwordHash, "wrong horse").Return(false)
				f.expectUpdate(f.store.users, userFixture(f.t))
			},
			check: func(t *testing.T, saved *domain.User) {
				if got := saved.FailedLogins(); got != 1 {
					t.Errorf("ChangePassword saved FailedLogins = %d, want 1", got)
				}
				if got := saved.PasswordHash(); got != passwordHash {
					t.Errorf("ChangePassword saved PasswordHash = %q, want %q", got, passwordHash)
				}
			},
			wantErr:    domain.ErrInvalidCredentials,
			wantEvents: []loggedEvent{{"WARN", "login_failed", userID}},
		},
		{
			name:  "the fifth wrong current password locks the account",
			input: inputWith(func(input *usecase.ChangePasswordInput) { input.Current = "wrong horse" }),
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(failedFixture(f.t, 4), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(false)
				f.expectUpdate(f.store.users, failedFixture(f.t, 4))
			},
			check: func(t *testing.T, saved *domain.User) {
				if got := saved.Locked(baseTime); !got {
					t.Errorf("ChangePassword saved Locked(baseTime) = %v, want true", got)
				}
			},
			wantErr:    domain.ErrInvalidCredentials,
			wantEvents: []loggedEvent{{"WARN", "login_failed", userID}, {"WARN", "account_locked", userID}},
		},
		{
			name:  "a wrong current password saved on an account locked meanwhile is ErrAccountLocked",
			input: inputWith(func(input *usecase.ChangePasswordInput) { input.Current = "wrong horse" }),
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(false)
				f.expectUpdate(f.store.users, lockedFixture(f.t))
			},
			check: func(t *testing.T, saved *domain.User) {
				if got := saved.PasswordHash(); got != passwordHash {
					t.Errorf("ChangePassword saved PasswordHash = %q, want %q", got, passwordHash)
				}
			},
			wantErr:    domain.ErrAccountLocked,
			wantEvents: []loggedEvent{{"WARN", "login_failed", userID}},
		},
		{
			name:  "a right current password on an account locked meanwhile is refused before the new one is hashed",
			input: validInput,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(passwordHash, plainPassword).Return(true)
				f.expectUpdate(f.store.users, lockedFixture(f.t))
			},
			wantErr: domain.ErrAccountLocked,
		},
		{
			name:  "a lock check that cannot be made fails the call before the new password is hashed",
			input: validInput,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(passwordHash, plainPassword).Return(true)
				f.store.users.EXPECT().Update(gomock.Any(), userID, gomock.Any()).Return(nil, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:  "a failed attempt that cannot be saved fails the call",
			input: inputWith(func(input *usecase.ChangePasswordInput) { input.Current = "wrong horse" }),
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(userFixture(f.t), nil)
				f.hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(false)
				f.store.users.EXPECT().Update(gomock.Any(), userID, gomock.Any()).Return(nil, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:  "a locked account cannot change its password",
			input: validInput,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(lockedFixture(f.t), nil)
			},
			wantErr: domain.ErrAccountLocked,
		},
		{
			name:  "a current password of the wrong length is ErrInvalidCredentials and counts nothing",
			input: inputWith(func(input *usecase.ChangePasswordInput) { input.Current = strings.Repeat("a", 73) }),
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(userFixture(f.t), nil)
			},
			wantErr:    domain.ErrInvalidCredentials,
			wantEvents: []loggedEvent{{"WARN", "login_failed", userID}},
		},
		{
			name:    "a weak new password is refused before the store is called",
			input:   inputWith(func(input *usecase.ChangePasswordInput) { input.New = "short" }),
			mock:    func(*fixture) {},
			wantErr: domain.ErrWeakPassword,
		},
		{
			name:  "an unknown user is not found",
			input: validInput,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(nil, domain.ErrNotFound)
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name:  "a password that cannot be hashed fails the call and changes nothing",
			input: validInput,
			mock: func(f *fixture) {
				expectCurrentRight(f)
				f.hasher.EXPECT().Hash(gomock.Any()).Return("", errHashing)
			},
			wantErr: errHashing,
		},
		{
			name:  "an access token that cannot be signed changes nothing",
			input: validInput,
			mock: func(f *fixture) {
				expectCurrentRight(f)
				f.hasher.EXPECT().Hash(gomock.Any()).Return(newPasswordHash, nil)
				f.tokens.EXPECT().Issue(gomock.Any(), gomock.Any()).Return("", time.Time{}, errSigning)
			},
			wantErr: errSigning,
		},
		{
			name:  "a lock taken after the new password is hashed is honoured when the change is saved",
			input: validInput,
			mock: func(f *fixture) {
				expectCurrentRight(f)
				f.hasher.EXPECT().Hash(gomock.Any()).Return(newPasswordHash, nil)
				f.tokens.EXPECT().Issue(gomock.Any(), gomock.Any()).Return(accessToken, accessExpiry, nil)
				f.expectUnitOfWork()
				f.expectUpdate(f.tx.users, lockedFixture(f.t))
			},
			wantErr: domain.ErrAccountLocked,
		},
		{
			name:  "a user who cannot be saved fails the call with no pair",
			input: validInput,
			mock: func(f *fixture) {
				expectCurrentRight(f)
				f.hasher.EXPECT().Hash(gomock.Any()).Return(newPasswordHash, nil)
				f.tokens.EXPECT().Issue(gomock.Any(), gomock.Any()).Return(accessToken, accessExpiry, nil)
				f.expectUnitOfWork()
				f.tx.users.EXPECT().Update(gomock.Any(), userID, gomock.Any()).Return(nil, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:  "sessions that cannot be revoked fail the call with no pair",
			input: validInput,
			mock: func(f *fixture) {
				expectCurrentRight(f)
				f.hasher.EXPECT().Hash(gomock.Any()).Return(newPasswordHash, nil)
				f.tokens.EXPECT().Issue(gomock.Any(), gomock.Any()).Return(accessToken, accessExpiry, nil)
				f.expectUnitOfWork()
				f.expectUpdate(f.tx.users, userFixture(f.t))
				f.tx.sessions.EXPECT().RevokeAll(gomock.Any(), userID, baseTime).Return(errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:  "a session that cannot be stored fails the call with no pair",
			input: validInput,
			mock: func(f *fixture) {
				expectCurrentRight(f)
				f.hasher.EXPECT().Hash(gomock.Any()).Return(newPasswordHash, nil)
				f.tokens.EXPECT().Issue(gomock.Any(), gomock.Any()).Return(accessToken, accessExpiry, nil)
				f.expectUnitOfWork()
				f.expectUpdate(f.tx.users, userFixture(f.t))
				f.tx.sessions.EXPECT().RevokeAll(gomock.Any(), userID, baseTime).Return(nil)
				f.tx.sessions.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errStoreDown)
			},
			wantErr: errStoreDown,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			got, err := f.useCase.ChangePassword(t.Context(), tc.input)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ChangePassword = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ChangePassword = %+v, want %+v", got, tc.want)
			}
			if got := f.securityEvents(t); !reflect.DeepEqual(got, tc.wantEvents) {
				t.Errorf("ChangePassword logged %+v, want %+v", got, tc.wantEvents)
			}
			if tc.check != nil {
				if f.updated == nil {
					t.Fatalf("ChangePassword saved user = nil, want a user")
				}
				tc.check(t, f.updated)
			}
		})
	}
}
