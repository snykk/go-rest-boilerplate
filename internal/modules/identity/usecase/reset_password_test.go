package usecase_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// ResetPassword changes the password, ends any lockout, spends the token and revokes every session of the user
// in one unit of work, and refuses an unknown, expired or used token with ErrInvalidToken, an unknown one before
// the new password is hashed. Only a committed reset is logged as a security event, with the user's ID.
func TestResetPassword(t *testing.T) {
	expectKnown := func(f *fixture) *gomock.Call {
		return f.store.resets.EXPECT().ByHash(gomock.Any(), sha(randomToken)).Return(resetFor(userID), nil)
	}
	expectHash := func(f *fixture) *gomock.Call {
		return f.hasher.EXPECT().Hash(newPlainPassword).Return(newPasswordHash, nil)
	}
	expectLookup := func(f *fixture) {
		f.tx.resets.EXPECT().ByHash(gomock.Any(), sha(randomToken)).Return(resetFor(userID), nil)
	}
	expectSecret := func(f *fixture, change func(secret *domain.Secret)) {
		f.expectUse(f.tx.resets, secretWith(resetFor(userID), change))
	}
	tCases := []struct {
		name        string
		newPassword string
		mock        func(f *fixture)
		check       func(t *testing.T, saved *domain.User)
		wantErr     error
		// wantEvents are the security events logged, in order; none when nil.
		wantEvents []loggedEvent
	}{
		{
			name:        "the password, the token and every session change in one unit of work",
			newPassword: newPlainPassword,
			mock: func(f *fixture) {
				known := expectKnown(f)
				hash := expectHash(f)
				work := f.expectUnitOfWork()
				lookup := f.tx.resets.EXPECT().ByHash(gomock.Any(), sha(randomToken)).Return(resetFor(userID), nil)
				use := f.expectUse(f.tx.resets, resetFor(userID))
				update := f.expectUpdate(f.tx.users, userFixture(f.t))
				revoke := f.tx.sessions.EXPECT().RevokeAll(gomock.Any(), userID, baseTime).Return(nil)
				gomock.InOrder(known, hash, work, lookup, use, update, revoke)
			},
			check: func(t *testing.T, saved *domain.User) {
				if got := saved.PasswordHash(); got != newPasswordHash {
					t.Errorf("ResetPassword saved PasswordHash = %q, want %q", got, newPasswordHash)
				}
			},
			wantEvents: []loggedEvent{{"INFO", "password_reset", userID}},
		},
		{
			name:        "a reset ends the lockout of a locked account",
			newPassword: newPlainPassword,
			mock: func(f *fixture) {
				expectKnown(f)
				expectHash(f)
				f.expectUnitOfWork()
				expectLookup(f)
				f.expectUse(f.tx.resets, resetFor(userID))
				f.expectUpdate(f.tx.users, lockedFixture(f.t))
				f.tx.sessions.EXPECT().RevokeAll(gomock.Any(), userID, baseTime).Return(nil)
			},
			check: func(t *testing.T, saved *domain.User) {
				_, lockedBefore := saved.LockedUntil()
				if saved.PasswordHash() != newPasswordHash || saved.FailedLogins() != 0 || saved.Locked(baseTime) ||
					lockedBefore {
					t.Errorf("ResetPassword saved hash %q, %d failures, locked %t, a lockout on record %t; "+
						"want %q, 0, false, false", saved.PasswordHash(), saved.FailedLogins(), saved.Locked(baseTime),
						lockedBefore, newPasswordHash)
				}
			},
			wantEvents: []loggedEvent{{"INFO", "password_reset", userID}},
		},
		{
			name:        "an unknown token is ErrInvalidToken before the password is hashed",
			newPassword: newPlainPassword,
			mock: func(f *fixture) {
				f.store.resets.EXPECT().ByHash(gomock.Any(), sha(randomToken)).
					Return(domain.Secret{}, domain.ErrInvalidToken)
			},
			wantErr: domain.ErrInvalidToken,
		},
		{
			name:        "a token gone by the time of the unit of work is ErrInvalidToken",
			newPassword: newPlainPassword,
			mock: func(f *fixture) {
				expectKnown(f)
				expectHash(f)
				f.expectUnitOfWork()
				f.tx.resets.EXPECT().ByHash(gomock.Any(), sha(randomToken)).
					Return(domain.Secret{}, domain.ErrInvalidToken)
			},
			wantErr: domain.ErrInvalidToken,
		},
		{
			name:        "a store that cannot be reached fails the call before the password is hashed",
			newPassword: newPlainPassword,
			mock: func(f *fixture) {
				f.store.resets.EXPECT().ByHash(gomock.Any(), sha(randomToken)).Return(domain.Secret{}, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:        "an expired token is ErrInvalidToken and changes nothing",
			newPassword: newPlainPassword,
			mock: func(f *fixture) {
				expectKnown(f)
				expectHash(f)
				f.expectUnitOfWork()
				expectLookup(f)
				expectSecret(f, func(secret *domain.Secret) { secret.ExpiresAt = baseTime.Add(-time.Second) })
			},
			wantErr: domain.ErrInvalidToken,
		},
		{
			name:        "a token already guessed at is ErrInvalidToken",
			newPassword: newPlainPassword,
			mock: func(f *fixture) {
				expectKnown(f)
				expectHash(f)
				f.expectUnitOfWork()
				expectLookup(f)
				expectSecret(f, func(secret *domain.Secret) { secret.Attempts = 1 })
			},
			wantErr: domain.ErrInvalidToken,
		},
		{
			name:        "a token replaced since it was found is ErrInvalidToken",
			newPassword: newPlainPassword,
			mock: func(f *fixture) {
				expectKnown(f)
				expectHash(f)
				f.expectUnitOfWork()
				expectLookup(f)
				expectSecret(f, func(secret *domain.Secret) { secret.Hash = sha("other") })
			},
			wantErr: domain.ErrInvalidToken,
		},
		{
			name:        "a token used up between the lookup and the check is ErrInvalidToken",
			newPassword: newPlainPassword,
			mock: func(f *fixture) {
				expectKnown(f)
				expectHash(f)
				f.expectUnitOfWork()
				expectLookup(f)
				f.tx.resets.EXPECT().Use(gomock.Any(), userID, gomock.Any()).Return(domain.ErrInvalidCode)
			},
			wantErr: domain.ErrInvalidToken,
		},
		{
			name:        "a token that cannot be used fails the call",
			newPassword: newPlainPassword,
			mock: func(f *fixture) {
				expectKnown(f)
				expectHash(f)
				f.expectUnitOfWork()
				expectLookup(f)
				f.tx.resets.EXPECT().Use(gomock.Any(), userID, gomock.Any()).Return(errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:        "a weak new password is refused before the token is looked up",
			newPassword: "short",
			mock:        func(*fixture) {},
			wantErr:     domain.ErrWeakPassword,
		},
		{
			name:        "a password that cannot be hashed fails the call before the store is called",
			newPassword: newPlainPassword,
			mock: func(f *fixture) {
				expectKnown(f)
				f.hasher.EXPECT().Hash(gomock.Any()).Return("", errHashing)
			},
			wantErr: errHashing,
		},
		{
			name:        "a user who cannot be saved fails the call and revokes no session",
			newPassword: newPlainPassword,
			mock: func(f *fixture) {
				expectKnown(f)
				expectHash(f)
				f.expectUnitOfWork()
				expectLookup(f)
				f.expectUse(f.tx.resets, resetFor(userID))
				f.tx.users.EXPECT().Update(gomock.Any(), userID, gomock.Any()).Return(nil, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:        "sessions that cannot be revoked fail the call",
			newPassword: newPlainPassword,
			mock: func(f *fixture) {
				expectKnown(f)
				expectHash(f)
				f.expectUnitOfWork()
				expectLookup(f)
				f.expectUse(f.tx.resets, resetFor(userID))
				f.expectUpdate(f.tx.users, userFixture(f.t))
				f.tx.sessions.EXPECT().RevokeAll(gomock.Any(), userID, baseTime).Return(errStoreDown)
			},
			wantErr: errStoreDown,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			err := f.useCase.ResetPassword(t.Context(), randomToken, tc.newPassword)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ResetPassword = %v, want %v", err, tc.wantErr)
			}
			if got := f.securityEvents(t); !reflect.DeepEqual(got, tc.wantEvents) {
				t.Errorf("ResetPassword logged %+v, want %+v", got, tc.wantEvents)
			}
			if tc.check != nil {
				if f.updated == nil {
					t.Fatalf("ResetPassword saved user = nil, want a user")
				}
				tc.check(t, f.updated)
			}
		})
	}
}
