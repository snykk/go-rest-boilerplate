package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// ResendVerification mails a new code to an unverified user, unless the code the user holds is younger than
// the cooldown, and answers nil for an email it cannot send to, so the answer does not tell whether an account
// exists or a code was sent.
func TestResendVerification(t *testing.T) {
	tCases := []struct {
		name    string
		email   string
		mock    func(f *fixture)
		wantErr error
	}{
		{
			name:  "an unverified user is mailed a new code",
			email: userEmail,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), emailFixture(f.t)).Return(unverifiedFixture(f.t), nil)
				f.store.verifications.EXPECT().Put(gomock.Any(), codeFor(userID), settings().ResendCooldown).
					Return(true, nil)
				f.mail.EXPECT().Verification(gomock.Any(), emailFixture(f.t), verifyCode)
			},
		},
		{
			name:  "the email is normalized before the lookup",
			email: "  Ada@Example.COM ",
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), emailFixture(f.t)).Return(unverifiedFixture(f.t), nil)
				f.store.verifications.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				f.mail.EXPECT().Verification(gomock.Any(), emailFixture(f.t), verifyCode)
			},
		},
		{
			name:  "a code issued within the cooldown is kept, and nil is answered with nothing mailed",
			email: userEmail,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(unverifiedFixture(f.t), nil)
				f.store.verifications.EXPECT().Put(gomock.Any(), codeFor(userID), settings().ResendCooldown).
					Return(false, nil)
			},
		},
		{
			name:  "an unknown email answers nil and mails nothing",
			email: userEmail,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
			},
		},
		{
			name:  "an invalid email answers nil and mails nothing",
			email: "ada",
			mock:  func(*fixture) {},
		},
		{
			name:  "a verified user answers nil and mails nothing",
			email: userEmail,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(userFixture(f.t), nil)
			},
		},
		{
			name:  "a store that cannot be reached fails the call",
			email: userEmail,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(nil, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name:  "a code that cannot be stored fails the call and mails nothing",
			email: userEmail,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(unverifiedFixture(f.t), nil)
				f.store.verifications.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, errStoreDown)
			},
			wantErr: errStoreDown,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			err := f.useCase.ResendVerification(t.Context(), tc.email)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ResendVerification = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// ResendVerification keeps the cooldown after the code was guessed at too often. The store keeps the exhausted
// code, as SecretRepository.Use promises, so a resend within the cooldown stores and mails nothing, and the
// right code is still refused afterwards.
func TestResendVerificationAfterExhaustion(t *testing.T) {
	f := newFixture(t)
	held := codeFor(userID)
	f.store.users.EXPECT().ByEmail(gomock.Any(), emailFixture(f.t)).Return(unverifiedFixture(f.t), nil).AnyTimes()
	f.store.verifications.EXPECT().Use(gomock.Any(), userID, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, check func(*domain.Secret) error) error {
			secret := held
			err := check(&secret)
			if errors.Is(err, domain.ErrInvalidCode) {
				held = secret
			}
			return err
		}).AnyTimes()
	f.store.verifications.EXPECT().Put(gomock.Any(), gomock.Any(), settings().ResendCooldown).DoAndReturn(
		func(_ context.Context, next domain.Secret, cooldown time.Duration) (bool, error) {
			if held.CreatedAt.After(next.CreatedAt.Add(-cooldown)) {
				return false, nil
			}
			held = next
			return true, nil
		})
	for range settings().CodeAttempts {
		if err := f.useCase.Verify(t.Context(), userEmail, "000000"); !errors.Is(err, domain.ErrInvalidCode) {
			t.Fatalf("Verify with a wrong code = %v, want %v", err, domain.ErrInvalidCode)
		}
	}
	if err := f.useCase.Verify(t.Context(), userEmail, verifyCode); !errors.Is(err, domain.ErrTooManyAttempts) {
		t.Fatalf("Verify with the right code after the wrong ones = %v, want %v", err, domain.ErrTooManyAttempts)
	}

	err := f.useCase.ResendVerification(t.Context(), userEmail)

	if err != nil {
		t.Fatalf("ResendVerification = %v, want nil", err)
	}
	if err := f.useCase.Verify(t.Context(), userEmail, verifyCode); !errors.Is(err, domain.ErrTooManyAttempts) {
		t.Errorf("Verify with the right code after the resend = %v, want %v", err, domain.ErrTooManyAttempts)
	}
}
