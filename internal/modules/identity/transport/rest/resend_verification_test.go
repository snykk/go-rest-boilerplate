package rest_test

import (
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"
)

// ResendVerification answers 202 with no body whatever the email, so the answer does not tell whether an
// account exists. It refuses a body without an email before the use cases are called, and answers a store
// failure with 500.
func TestHandlerResendVerification(t *testing.T) {
	tCases := []struct {
		name string
		body string
		mock func(f *fixture)
		want outcome
	}{
		{
			name: "an email is accepted",
			body: `{"email":"ada@example.com"}`,
			mock: func(f *fixture) { f.useCases.EXPECT().ResendVerification(gomock.Any(), userEmail).Return(nil) },
			want: outcome{status: 202},
		},
		{
			name: "a missing email is refused before the use cases are called",
			body: `{}`,
			want: outcome{status: 422, code: "validation_failed", field: "email"},
		},
		{
			name: "a store failure is an internal error that does not leak its cause",
			body: `{"email":"ada@example.com"}`,
			mock: func(f *fixture) {
				f.useCases.EXPECT().ResendVerification(gomock.Any(), userEmail).Return(errStoreDown)
			},
			want: outcome{status: 500, code: "internal", logged: "10.0.0.1"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.mock != nil {
				tc.mock(f)
			}

			recorder := f.serve(http.MethodPost, "/v1/auth/verification/resend", tc.body)

			f.check(t, recorder, tc.want)
		})
	}
}
