package rest_test

import (
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

// ChangePassword changes the password of the signed-in caller and answers 200 with a new token pair, which no
// cache may keep. It maps a wrong current password, a locked account, a caller who is gone and a weak new
// password to their problems, the last naming the new_password field; it never repeats either password, and
// answers 401 to a request with no caller.
func TestHandlerChangePassword(t *testing.T) {
	const body = `{"current_password":"` + secretPassword + `","new_password":"` + newSecretPassword + `"}`
	expectChangeFails := func(err error) func(f *fixture) {
		return func(f *fixture) {
			f.useCases.EXPECT().ChangePassword(gomock.Any(), gomock.Any()).Return(usecase.TokenPair{}, err)
		}
	}
	tCases := []struct {
		name   string
		target string // "" means /v1/auth/password/change, whose requests come from userID
		body   string
		mock   func(f *fixture)
		want   outcome
	}{
		{
			name: "the caller's password is changed",
			body: body,
			mock: func(f *fixture) {
				f.useCases.EXPECT().ChangePassword(gomock.Any(), usecase.ChangePasswordInput{
					UserID:  userID,
					Current: secretPassword,
					New:     newSecretPassword,
				}).Return(pairFixture(), nil)
			},
			want: outcome{status: 200, body: pairJSON, noStore: true},
		},
		{
			name:   "a request with no caller is unauthenticated",
			target: "/v2/auth/password/change",
			body:   body,
			want:   outcome{status: 401, code: "unauthenticated"},
		},
		{
			name: "a missing new password is refused before the use cases are called",
			body: `{"current_password":"` + secretPassword + `"}`,
			want: outcome{status: 422, code: "validation_failed", field: "new_password"},
		},
		{
			name: "a wrong current password is unauthorized",
			body: body,
			mock: expectChangeFails(domain.ErrInvalidCredentials),
			want: outcome{status: 401, code: "invalid_credentials"},
		},
		{
			name: "a locked account is locked",
			body: body,
			mock: expectChangeFails(domain.ErrAccountLocked),
			want: outcome{status: 423, code: "account_locked"},
		},
		{
			name: "a weak new password names the new_password field",
			body: body,
			mock: expectChangeFails(domain.ErrWeakPassword),
			want: outcome{status: 422, code: "validation_failed", field: "new_password"},
		},
		{
			name: "a caller who is gone is not found",
			body: body,
			mock: expectChangeFails(domain.ErrNotFound),
			want: outcome{status: 404, code: "not_found"},
		},
		{
			name: "an unexpected error is an internal error that does not leak its cause",
			body: body,
			mock: expectChangeFails(errStoreDown),
			want: outcome{status: 500, code: "internal", logged: "10.0.0.1"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.mock != nil {
				tc.mock(f)
			}
			target := tc.target
			if target == "" {
				target = "/v1/auth/password/change"
			}

			recorder := f.serve(http.MethodPost, target, tc.body)

			f.check(t, recorder, tc.want)
		})
	}
}
