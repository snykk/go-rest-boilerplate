package rest_test

import (
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
)

// Verify answers 204 with no body when the code is right, and maps a wrong, expired or exhausted code to its
// problem. It refuses a body without a code before the use cases are called.
func TestHandlerVerify(t *testing.T) {
	const body = `{"email":"ada@example.com","code":"` + verifyCode + `"}`
	expectVerify := func(err error) func(f *fixture) {
		return func(f *fixture) { f.useCases.EXPECT().Verify(gomock.Any(), userEmail, verifyCode).Return(err) }
	}
	tCases := []struct {
		name string
		body string
		mock func(f *fixture)
		want outcome
	}{
		{name: "the right code verifies the email", body: body, mock: expectVerify(nil), want: outcome{status: 204}},
		{
			name: "a missing code is refused before the use cases are called",
			body: `{"email":"ada@example.com"}`,
			want: outcome{status: 422, code: "validation_failed", field: "code"},
		},
		{
			name: "a wrong code is unprocessable",
			body: body,
			mock: expectVerify(domain.ErrInvalidCode),
			want: outcome{status: 422, code: "invalid_code"},
		},
		{
			name: "an expired code is unprocessable",
			body: body,
			mock: expectVerify(domain.ErrCodeExpired),
			want: outcome{status: 422, code: "code_expired"},
		},
		{
			name: "a code guessed at too often is unprocessable",
			body: body,
			mock: expectVerify(domain.ErrTooManyAttempts),
			want: outcome{status: 422, code: "too_many_attempts"},
		},
		{
			name: "an unexpected error is an internal error that does not leak its cause",
			body: body,
			mock: expectVerify(errStoreDown),
			want: outcome{status: 500, code: "internal", logged: "10.0.0.1"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.mock != nil {
				tc.mock(f)
			}

			recorder := f.serve(http.MethodPost, "/v1/auth/verification", tc.body)

			f.check(t, recorder, tc.want)
		})
	}
}
