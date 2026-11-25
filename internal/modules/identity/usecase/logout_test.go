package usecase_test

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

// Logout revokes the session family of the refresh token it is given, by the hash of the token, and fails only
// when the store does.
func TestLogout(t *testing.T) {
	tCases := []struct {
		name    string
		token   string
		mock    func(f *fixture)
		wantErr error
	}{
		{
			name:  "the family of the token is revoked by the hash of the token",
			token: oldRefreshToken,
			mock: func(f *fixture) {
				f.store.sessions.EXPECT().Revoke(gomock.Any(), sha(oldRefreshToken), baseTime).Return(nil)
			},
		},
		{
			name:  "a store that cannot be reached fails the call",
			token: oldRefreshToken,
			mock: func(f *fixture) {
				f.store.sessions.EXPECT().Revoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(errStoreDown)
			},
			wantErr: errStoreDown,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			err := f.useCase.Logout(t.Context(), tc.token)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Logout = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
