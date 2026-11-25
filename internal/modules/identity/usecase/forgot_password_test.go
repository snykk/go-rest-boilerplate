package usecase_test

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// ForgotPassword stores a reset token and mails it for a known, verified user, unless the token the user holds
// is younger than the cooldown, and answers nil for any other email, so the answer does not tell whether an
// account exists or a token was sent.
func TestForgotPassword(t *testing.T) {
	tCases := []struct {
		name    string
		email   string
		mock    func(f *fixture)
		wantErr error
	}{
		{
			name:  "a verified user is mailed a reset token",
			email: userEmail,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), emailFixture(f.t)).Return(userFixture(f.t), nil)
				f.store.resets.EXPECT().Put(gomock.Any(), resetFor(userID), settings().ResendCooldown).Return(true, nil)
				f.mail.EXPECT().PasswordReset(gomock.Any(), emailFixture(f.t), randomToken)
			},
		},
		{
			name:  "the email is normalized before the lookup",
			email: "  Ada@Example.COM ",
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), emailFixture(f.t)).Return(userFixture(f.t), nil)
				f.store.resets.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				f.mail.EXPECT().PasswordReset(gomock.Any(), emailFixture(f.t), randomToken)
			},
		},
		{
			name:  "a token issued within the cooldown is kept, and nil is answered with nothing mailed",
			email: userEmail,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(userFixture(f.t), nil)
				f.store.resets.EXPECT().Put(gomock.Any(), resetFor(userID), settings().ResendCooldown).
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
			name:  "an unverified user answers nil and mails nothing",
			email: userEmail,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(unverifiedFixture(f.t), nil)
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
			name:  "a token that cannot be stored fails the call and mails nothing",
			email: userEmail,
			mock: func(f *fixture) {
				f.store.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(userFixture(f.t), nil)
				f.store.resets.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, errStoreDown)
			},
			wantErr: errStoreDown,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			err := f.useCase.ForgotPassword(t.Context(), tc.email)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ForgotPassword = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
