package usecase_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

// Refresh rotates a live refresh token into a new pair, with the roles the user holds now, and refuses with
// ErrInvalidToken what the store refuses, issuing no token. A token the store reports as reused is logged as a
// security event with its user, and refused the same way.
func TestRefresh(t *testing.T) {
	expectRotate := func(f *fixture) *gomock.Call {
		next := domain.Session{
			ID:        "id-1",
			TokenHash: sha(randomToken),
			ExpiresAt: baseTime.Add(settings().RefreshTTL),
			CreatedAt: baseTime,
		}
		return f.store.sessions.EXPECT().Rotate(
			gomock.Any(), sha(oldRefreshToken), next, baseTime, settings().ReuseGrace)
	}
	editor := func(stored *domain.UserStored) { stored.Roles = []string{"reader", "editor"} }
	tCases := []struct {
		name    string
		mock    func(f *fixture)
		want    usecase.TokenPair
		wantErr error
		// wantEvents are the security events logged, in order; none when nil.
		wantEvents []loggedEvent
	}{
		{
			name: "a valid token is rotated into a new pair",
			mock: func(f *fixture) {
				expectRotate(f).Return(userID, nil)
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(userFixture(f.t), nil)
				f.tokens.EXPECT().Issue(userID, []string{"reader"}).Return(accessToken, accessExpiry, nil)
			},
			want: pairFixture(),
		},
		{
			name: "the new access token carries the roles the user holds now",
			mock: func(f *fixture) {
				expectRotate(f).Return(userID, nil)
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(userWith(f.t, editor), nil)
				f.tokens.EXPECT().Issue(userID, []string{"reader", "editor"}).Return(accessToken, accessExpiry, nil)
			},
			want: pairFixture(),
		},
		{
			name: "a token Rotate refuses is ErrInvalidToken and issues nothing",
			mock: func(f *fixture) {
				expectRotate(f).Return("", domain.ErrInvalidToken)
			},
			wantErr: domain.ErrInvalidToken,
		},
		{
			name: "a token used again after the grace is logged as a reuse and is ErrInvalidToken",
			mock: func(f *fixture) {
				expectRotate(f).Return(userID, domain.ErrTokenReused)
			},
			wantErr:    domain.ErrInvalidToken,
			wantEvents: []loggedEvent{{"WARN", "refresh_reused", userID}},
		},
		{
			name: "a store that cannot be reached fails the call",
			mock: func(f *fixture) {
				expectRotate(f).Return("", errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name: "a user who is gone is ErrInvalidToken",
			mock: func(f *fixture) {
				expectRotate(f).Return(userID, nil)
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(nil, domain.ErrNotFound)
			},
			wantErr: domain.ErrInvalidToken,
		},
		{
			name: "a user who cannot be loaded fails the call",
			mock: func(f *fixture) {
				expectRotate(f).Return(userID, nil)
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(nil, errStoreDown)
			},
			wantErr: errStoreDown,
		},
		{
			name: "an access token that cannot be signed fails the call with no pair",
			mock: func(f *fixture) {
				expectRotate(f).Return(userID, nil)
				f.store.users.EXPECT().ByID(gomock.Any(), userID).Return(userFixture(f.t), nil)
				f.tokens.EXPECT().Issue(gomock.Any(), gomock.Any()).Return("", time.Time{}, errSigning)
			},
			wantErr: errSigning,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mock(f)

			got, err := f.useCase.Refresh(t.Context(), oldRefreshToken)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Refresh = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Refresh = %+v, want %+v", got, tc.want)
			}
			if got := f.securityEvents(t); !reflect.DeepEqual(got, tc.wantEvents) {
				t.Errorf("Refresh logged %+v, want %+v", got, tc.wantEvents)
			}
		})
	}
}
