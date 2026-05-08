package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	authuc "github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	authhandler "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1/auth"
	"github.com/snykk/go-rest-boilerplate/internal/test/mocks"
	jwtpkg "github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
	_ = validators.ValidatePayloads(struct{}{})
}

type authHarness struct {
	uc     *mocks.AuthUsecase
	router *gin.Engine
}

func newAuthHarness(t *testing.T) authHarness {
	t.Helper()
	uc := mocks.NewAuthUsecase(t)
	h := authhandler.NewHandler(uc)
	r := gin.New()
	r.POST("/login", h.Login)
	r.POST("/register", h.Register)
	r.POST("/password/forgot", h.ForgotPassword)
	r.POST("/password/reset", h.ResetPassword)
	r.PUT("/password/change", injectClaims("user-1", "patrick@example.com"), h.ChangePassword)
	return authHarness{uc: uc, router: r}
}

func injectClaims(userID, email string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(constants.CtxAuthenticatedUserKey, jwtpkg.JwtCustomClaim{
			UserID: userID,
			Email:  email,
		})
		c.Next()
	}
}

func doJSON(t *testing.T, h authHarness, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

func TestLoginHandler(t *testing.T) {
	t.Run("happy path returns 200 and tokens", func(t *testing.T) {
		h := newAuthHarness(t)
		h.uc.On("Login", mock.Anything, authuc.LoginRequest{Email: "patrick@example.com", Password: "Pwd_123!"}).
			Return(authuc.LoginResponse{
				AccessToken: "access-tok", RefreshToken: "refresh-tok",
			}, nil).Once()

		w := doJSON(t, h, "POST", "/login", map[string]string{
			"email": "patrick@example.com", "password": "Pwd_123!",
		})
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "access-tok")
	})

	t.Run("malformed JSON returns 400", func(t *testing.T) {
		h := newAuthHarness(t)
		req := httptest.NewRequest("POST", "/login", bytes.NewBufferString("{bad json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation failure returns 422", func(t *testing.T) {
		h := newAuthHarness(t)
		w := doJSON(t, h, "POST", "/login", map[string]string{
			"email": "not-an-email", "password": "p",
		})
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("usecase Unauthorized returns 401", func(t *testing.T) {
		h := newAuthHarness(t)
		h.uc.On("Login", mock.Anything, authuc.LoginRequest{Email: "x@y.com", Password: "wrong"}).
			Return(authuc.LoginResponse{}, apperror.Unauthorized("invalid email or password")).Once()
		w := doJSON(t, h, "POST", "/login", map[string]string{
			"email": "x@y.com", "password": "wrong",
		})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestForgotPasswordHandler(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		h := newAuthHarness(t)
		h.uc.On("ForgotPassword", mock.Anything, authuc.ForgotPasswordRequest{Email: "patrick@example.com"}).Return(nil).Once()
		w := doJSON(t, h, "POST", "/password/forgot", map[string]string{"email": "patrick@example.com"})
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("infra error from usecase still returns non-2xx", func(t *testing.T) {
		h := newAuthHarness(t)
		h.uc.On("ForgotPassword", mock.Anything, authuc.ForgotPasswordRequest{Email: "patrick@example.com"}).
			Return(apperror.InternalCause(assertErr("redis down"))).Once()
		w := doJSON(t, h, "POST", "/password/forgot", map[string]string{"email": "patrick@example.com"})
		assert.GreaterOrEqual(t, w.Code, 500)
	})
}

func TestResetPasswordHandler(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		h := newAuthHarness(t)
		h.uc.On("ResetPassword", mock.Anything, authuc.ResetPasswordRequest{Token: "tok-1", NewPassword: "Newpwd_9999!"}).Return(nil).Once()
		w := doJSON(t, h, "POST", "/password/reset", map[string]string{
			"token": "tok-1", "new_password": "Newpwd_9999!",
		})
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		h := newAuthHarness(t)
		h.uc.On("ResetPassword", mock.Anything, authuc.ResetPasswordRequest{Token: "stale", NewPassword: "Newpwd_9999!"}).
			Return(apperror.Unauthorized("reset token is invalid or expired")).Once()
		w := doJSON(t, h, "POST", "/password/reset", map[string]string{
			"token": "stale", "new_password": "Newpwd_9999!",
		})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestChangePasswordHandler(t *testing.T) {
	t.Run("happy path returns 200 with claims injected", func(t *testing.T) {
		h := newAuthHarness(t)
		h.uc.On("ChangePassword", mock.Anything, authuc.ChangePasswordRequest{
			UserID:          "user-1",
			CurrentPassword: "Pwd_123!",
			NewPassword:     "Newpwd_9999!",
		}).Return(nil).Once()
		w := doJSON(t, h, "PUT", "/password/change", map[string]string{
			"current_password": "Pwd_123!", "new_password": "Newpwd_9999!",
		})
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("usecase Unauthorized when current password wrong", func(t *testing.T) {
		h := newAuthHarness(t)
		h.uc.On("ChangePassword", mock.Anything, authuc.ChangePasswordRequest{
			UserID:          "user-1",
			CurrentPassword: "wrong",
			NewPassword:     "Newpwd_9999!",
		}).Return(apperror.Unauthorized("current password is incorrect")).Once()
		w := doJSON(t, h, "PUT", "/password/change", map[string]string{
			"current_password": "wrong", "new_password": "Newpwd_9999!",
		})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func assertErr(s string) error { return &simpleErr{msg: s} }

type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }
