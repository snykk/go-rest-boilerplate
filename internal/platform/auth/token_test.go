package auth_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
)

// StaticToken accepts a request that carries its token as a bearer credential, in any case of the scheme,
// and refuses every other request, including every request when its token is empty.
func TestStaticToken(t *testing.T) {
	token := strings.Repeat("k", 32)
	operator := auth.Principal{ID: "operator", Roles: []string{"admin"}}
	tCases := []struct {
		name          string
		configured    string
		authorization string
		wantErr       error
	}{
		{name: "the token as a bearer credential is accepted", configured: token, authorization: "Bearer " + token},
		{name: "the scheme is matched in any case", configured: token, authorization: "bearer " + token},
		{
			name:          "another token is refused",
			configured:    token,
			authorization: "Bearer " + strings.Repeat("x", 32),
			wantErr:       auth.ErrUnauthenticated,
		},
		{
			name:          "the token in another scheme is refused",
			configured:    token,
			authorization: "Basic " + token,
			wantErr:       auth.ErrUnauthenticated,
		},
		{name: "a request without credentials is refused", configured: token, wantErr: auth.ErrUnauthenticated},
		{
			name:          "an empty token refuses even an empty credential",
			authorization: "Bearer ",
			wantErr:       auth.ErrUnauthenticated,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPut, "/admin/log-level", nil)
			if tc.authorization != "" {
				request.Header.Set("Authorization", tc.authorization)
			}

			got, err := auth.StaticToken(tc.configured, operator).Authenticate(request)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Authenticate = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && !reflect.DeepEqual(got, operator) {
				t.Errorf("Authenticate = %+v, want %+v", got, operator)
			}
		})
	}
}
