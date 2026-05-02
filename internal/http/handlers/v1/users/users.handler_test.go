package users_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	usershandler "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1/users"
	jwtpkg "github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/internal/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() { gin.SetMode(gin.TestMode) }

func TestGetUserDataHandler(t *testing.T) {
	build := func(t *testing.T) (*mocks.UsersUsecase, *gin.Engine) {
		uc := mocks.NewUsersUsecase(t)
		h := usershandler.NewHandler(uc)
		r := gin.New()
		r.GET("/me", func(c *gin.Context) {
			c.Set(constants.CtxAuthenticatedUserKey, jwtpkg.JwtCustomClaim{
				UserID: "user-1", Email: "patrick@example.com",
			})
			c.Next()
		}, h.GetUserData)
		return uc, r
	}

	t.Run("happy path returns user data", func(t *testing.T) {
		uc, r := build(t)
		uc.On("GetByEmail", mock.Anything, "patrick@example.com").
			Return(domain.User{ID: "user-1", Email: "patrick@example.com", Username: "patrick"}, nil).Once()

		req := httptest.NewRequest("GET", "/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "patrick")
	})

	t.Run("usecase NotFound surfaces as 404", func(t *testing.T) {
		uc, r := build(t)
		uc.On("GetByEmail", mock.Anything, "patrick@example.com").
			Return(domain.User{}, apperror.NotFound("user not found")).Once()

		req := httptest.NewRequest("GET", "/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("missing claims returns 401", func(t *testing.T) {
		uc := mocks.NewUsersUsecase(t)
		h := usershandler.NewHandler(uc)
		r := gin.New()
		r.GET("/me", h.GetUserData)

		req := httptest.NewRequest("GET", "/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
