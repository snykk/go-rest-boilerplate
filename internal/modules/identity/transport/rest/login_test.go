package rest_test

import (
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

// Login answers 200 with a token pair that no cache may keep, and maps a refused sign-in to its problem: wrong
// credentials, an unverified email or a locked account. It refuses a body without a password before the use
// cases are called, and never repeats the password.
func TestHandlerLogin(t *testing.T) {
	const body = `{"email":"ada@example.com","password":"` + secretPassword + `"}`
	expectLoginFails := func(err error) func(f *fixture) {
		return func(f *fixture) {
			f.useCases.EXPECT().Login(gomock.Any(), userEmail, secretPassword).Return(usecase.TokenPair{}, err)
		}
	}
	tCases := []struct {
		name string
		body string
		mock func(f *fixture)
		want outcome
	}{
		{
			name: "the right credentials sign in",
			body: body,
			mock: func(f *fixture) {
				f.useCases.EXPECT().Login(gomock.Any(), userEmail, secretPassword).Return(pairFixture(), nil)
			},
			want: outcome{status: 200, body: pairJSON, noStore: true},
		},
		{
			name: "a missing password is refused before the use cases are called",
			body: `{"email":"ada@example.com"}`,
			want: outcome{status: 422, code: "validation_failed", field: "password"},
		},
		{
			name: "wrong credentials are unauthorized, and the password is never repeated",
			body: body,
			mock: expectLoginFails(domain.ErrInvalidCredentials),
			want: outcome{status: 401, code: "invalid_credentials"},
		},
		{
			name: "an unverified email is forbidden",
			body: body,
			mock: expectLoginFails(domain.ErrEmailNotVerified),
			want: outcome{status: 403, code: "email_not_verified"},
		},
		{
			name: "a locked account is locked",
			body: body,
			mock: expectLoginFails(domain.ErrAccountLocked),
			want: outcome{status: 423, code: "account_locked"},
		},
		{
			name: "an unexpected error is an internal error that does not leak its cause",
			body: body,
			mock: expectLoginFails(errStoreDown),
			want: outcome{status: 500, code: "internal", logged: "10.0.0.1"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.mock != nil {
				tc.mock(f)
			}

			recorder := f.serve(http.MethodPost, "/v1/auth/login", tc.body)

			f.check(t, recorder, tc.want)
		})
	}
}
