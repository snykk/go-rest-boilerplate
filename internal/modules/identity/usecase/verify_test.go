package usecase_test

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// Verify marks the user's email verified when the code matches a live secret, and refuses a wrong, expired or
// exhausted code, and an email it does not know, with an error that does not tell which.
func TestVerify(t *testing.T) {
	tCases := []struct {
		name        string
		email       string
		code        string
		mock        func(f *fixture)
		wantErr     error
		wantUpdated bool
	}{
		{
			name:  "the right code verifies the user",
			email: userEmail,
			code:  verifyCode,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), emailFixture(f.t)).Return(unverifiedFixture(f.t), nil)
				f.expectUse(f.store.verifications, codeFor(userID))
				f.expectUpdate(f.store.users, unverifiedFixture(f.t))
			},
			wantUpdated: true,
		},
		{
			name:  "the email is normalized before the lookup",
			email: "  Ada@Example.COM ",
			code:  verifyCode,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), emailFixture(f.t)).Return(unverifiedFixture(f.t), nil)
				f.expectUse(f.store.verifications, codeFor(userID))
				f.expectUpdate(f.store.users, unverifiedFixture(f.t))
			},
			wantUpdated: true,
		},
		{
			name:  "a wrong code is refused and the user stays unverified",
			email: userEmail,
			code:  "000000",
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(unverifiedFixture(f.t), nil)
				f.expectUse(f.store.verifications, codeFor(userID))
			},
			wantErr: domain.ErrInvalidCode,
		},
		{
			name:  "an expired code is refused",
			email: userEmail,
			code:  verifyCode,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(unverifiedFixture(f.t), nil)
				f.expectUse(f.store.verifications, secretWith(codeFor(userID), func(secret *domain.Secret) {
					secret.ExpiresAt = baseTime.Add(-time.Second)
				}))
			},
			wantErr: domain.ErrCodeExpired,
		},
		{
			name:  "a code guessed at too often is refused even when it is right",
			email: userEmail,
			code:  verifyCode,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(unverifiedFixture(f.t), nil)
				f.expectUse(f.store.verifications, secretWith(codeFor(userID), func(secret *domain.Secret) {
					secret.Attempts = settings().CodeAttempts
				}))
			},
			wantErr: domain.ErrTooManyAttempts,
		},
		{
			name:  "a user with no code is refused",
			email: userEmail,
			code:  verifyCode,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(unverifiedFixture(f.t), nil)
				f.store.verifications.EXPECT().Use(gomock.Any(), userID, gomock.Any()).Return(domain.ErrInvalidCode)
			},
			wantErr: domain.ErrInvalidCode,
		},
		{
			name:  "an unknown email is refused as a wrong code",
			email: userEmail,
			code:  verifyCode,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
			},
			wantErr: domain.ErrInvalidCode,
		},
		{
			name:    "an invalid email is refused as a wrong code",
			email:   "ada",
			code:    verifyCode,
			mock:    func(*fixture) {},
			wantErr: domain.ErrInvalidCode,
		},
		{
			name:  "a store that cannot be reached fails the call",
			email: userEmail,
			code:  verifyCode,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(nil, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:  "a user that cannot be saved fails the call",
			email: userEmail,
			code:  verifyCode,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(unverifiedFixture(f.t), nil)
				f.expectUse(f.store.verifications, codeFor(userID))
				f.store.users.EXPECT().Update(gomock.Any(), userID, gomock.Any()).Return(nil, errStoreDown)
			},
			wantErr: errStoreDown,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			err := f.useCase.Verify(t.Context(), tc.email, tc.code)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Verify = %v, want %v", err, tc.wantErr)
			}
			if tc.wantUpdated {
				if f.updated == nil {
					t.Fatalf("Verify saved user = nil, want a user")
				}
				if got := f.updated.Verified(); !got {
					t.Errorf("Verify saved Verified() = %v, want true", got)
				}
			}
		})
	}
}
