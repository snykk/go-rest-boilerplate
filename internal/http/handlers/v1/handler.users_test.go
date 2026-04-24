package v1_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	golangJWT "github.com/golang-jwt/jwt/v5"
	"github.com/gin-gonic/gin"
	V1Domains "github.com/snykk/go-rest-boilerplate/internal/business/domains/v1"
	V1Usecases "github.com/snykk/go-rest-boilerplate/internal/business/usecases/v1"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	V1Handlers "github.com/snykk/go-rest-boilerplate/internal/http/handlers/v1"
	"github.com/snykk/go-rest-boilerplate/internal/mocks"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	jwtServiceMock  *mocks.JWTService
	userRepoMock    *mocks.UserRepository
	userUsecase     V1Domains.UserUsecase
	userHandler     V1Handlers.UserHandler
	mailerOTPMock   *mocks.OTPMailer
	redisMock       *mocks.RedisCache
	ristrettoMock   *mocks.RistrettoCache
	usersDataFromDB []V1Domains.UserDomain
	userDataFromDB  V1Domains.UserDomain
	s               *gin.Engine
)

func setup(t *testing.T) {
	// VerifyOTP reads OTP_MAX_ATTEMPTS and REDIS_EXPIRED from config.
	config.AppConfig.OTPMaxAttempts = 5
	config.AppConfig.REDISExpired = 5

	jwtServiceMock = mocks.NewJWTService(t)
	redisMock = mocks.NewRedisCache(t)
	mailerOTPMock = mocks.NewOTPMailer(t)
	ristrettoMock = mocks.NewRistrettoCache(t)
	userRepoMock = mocks.NewUserRepository(t)
	userUsecase = V1Usecases.NewUserUsecase(userRepoMock, jwtServiceMock, mailerOTPMock, redisMock, ristrettoMock)
	userHandler = V1Handlers.NewUserHandler(userUsecase)

	usersDataFromDB = []V1Domains.UserDomain{
		{
			ID:        "ddfcea5c-d919-4a8f-a631-4ace39337s3a",
			Username:  "itsmepatrick",
			Email:     "najibfikri13@gmail.com",
			RoleID:    1,
			Password:  "Test123!@",
			Active:    true,
			CreatedAt: time.Now(),
		},
		{
			ID:        "wifff3jd-idhd-0sis-8dua-4fiefie37kfj",
			Username:  "johny",
			Email:     "johny123@gmail.com",
			RoleID:    2,
			Password:  "Test123!@",
			Active:    true,
			CreatedAt: time.Now(),
		},
	}

	userDataFromDB = V1Domains.UserDomain{
		ID:        "fjskeie8-jfk8-qke0-sksj-ksjf89e8ehfu",
		Username:  "itsmepatrick",
		Email:     "najibfikri13@gmail.com",
		Password:  "Test123!@",
		RoleID:    2,
		Active:    false,
		CreatedAt: time.Now(),
	}

	// Create gin engine
	s = gin.Default()
	s.Use(lazyAuth)
}

func lazyAuth(ctx *gin.Context) {
	// prepare claims
	jwtClaims := jwt.JwtCustomClaim{
		UserID:  userDataFromDB.ID,
		IsAdmin: false,
		Email:   userDataFromDB.Email,
		RegisteredClaims: golangJWT.RegisteredClaims{
			ExpiresAt: golangJWT.NewNumericDate(time.Now().Add(time.Hour * 5)),
			Issuer:    userDataFromDB.Username,
			IssuedAt:  golangJWT.NewNumericDate(time.Now()),
		},
	}
	ctx.Set(constants.CtxAuthenticatedUserKey, jwtClaims)
}

func TestRegister(t *testing.T) {
	setup(t)
	s.POST(constants.EndpointV1+"/auth/register", userHandler.Register)
	t.Run("When Success Register", func(t *testing.T) {
		req := requests.UserRequest{
			Username: "itsmepatrick",
			Email:    "najibfikri13@gmail.com",
			Password: "Test123!@",
		}
		reqBody, _ := json.Marshal(req)

		userRepoMock.Mock.On("Store", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(nil).Once()
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/register", bytes.NewReader(reqBody))
		r.Header.Set("Content-Type", "application/json")

		s.ServeHTTP(w, r)

		body := w.Body.String()
		assert.Equal(t, http.StatusCreated, w.Result().StatusCode)
		assert.Contains(t, body, "registration user success")
	})
	t.Run("When Request is Empty", func(t *testing.T) {
		req := requests.UserRequest{}
		reqBody, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/register", bytes.NewReader(reqBody))
		r.Header.Set("Content-Type", "application/json")

		s.ServeHTTP(w, r)

		body := w.Body.String()
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
		assert.Contains(t, body, "required")
	})
}

func TestSendOTP(t *testing.T) {
	setup(t)
	s.POST(constants.EndpointV1+"/auth/send-otp", userHandler.SendOTP)
	t.Run("Test 1 | Success Send OTP", func(t *testing.T) {
		req := requests.UserSendOTPRequest{
			Email: "najibfikri13@gmail.com",
		}
		reqBody, _ := json.Marshal(req)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		mailerOTPMock.On("SendOTP", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Once()
		redisMock.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil).Once()
		redisMock.On("Del", mock.Anything, mock.AnythingOfType("string")).Return(nil).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/send-otp", bytes.NewReader(reqBody))
		r.Header.Set("Content-Type", "application/json")

		s.ServeHTTP(w, r)

		body := w.Body.String()
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Contains(t, body, "otp code has been send")
	})
	t.Run("Test 2 | Payloads is Empty", func(t *testing.T) {
		req := requests.UserSendOTPRequest{}
		reqBody, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/send-otp", bytes.NewReader(reqBody))
		r.Header.Set("Content-Type", "application/json")

		s.ServeHTTP(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})
	t.Run("Test 3 | When Failure Send OTP", func(t *testing.T) {
		req := requests.UserSendOTPRequest{
			Email: "najibfikri13@gmail.com",
		}
		reqBody, _ := json.Marshal(req)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		mailerOTPMock.On("SendOTP", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(constants.ErrUnexpected).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/send-otp", bytes.NewReader(reqBody))
		r.Header.Set("Content-Type", "application/json")

		s.ServeHTTP(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})
}

func TestVerifyOTP(t *testing.T) {
	setup(t)
	s.POST(constants.EndpointV1+"/auth/verify-otp", userHandler.VerifyOTP)
	t.Run("Test 1 | Success Verify OTP", func(t *testing.T) {
		req := requests.UserVerifOTPRequest{
			Email: "najibfikri13@gmail.com",
			Code:  "112233",
		}
		reqBody, _ := json.Marshal(req)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		redisMock.Mock.On("Incr", mock.Anything, mock.AnythingOfType("string")).Return(int64(1), nil).Once()
		redisMock.Mock.On("Expire", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil).Once()
		redisMock.Mock.On("Get", mock.Anything, mock.AnythingOfType("string")).Return("112233", nil).Once()
		userRepoMock.Mock.On("ChangeActiveUser", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(nil).Once()
		redisMock.On("Del", mock.Anything, mock.AnythingOfType("string")).Return(nil).Twice()
		ristrettoMock.On("Del", "users", mock.AnythingOfType("string")).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/verify-otp", bytes.NewReader(reqBody))
		r.Header.Set("Content-Type", "application/json")

		s.ServeHTTP(w, r)

		body := w.Body.String()
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Contains(t, body, "otp verification success")
	})
	t.Run("Test 2 | Payloads is Empty", func(t *testing.T) {
		req := requests.UserVerifOTPRequest{}
		reqBody, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/verify-otp", bytes.NewReader(reqBody))
		r.Header.Set("Content-Type", "application/json")

		s.ServeHTTP(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})
	t.Run("Test 3 | Invalid OTP Code", func(t *testing.T) {
		req := requests.UserVerifOTPRequest{
			Email: "najibfikri13@gmail.com",
			Code:  "999999",
		}
		reqBody, _ := json.Marshal(req)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		redisMock.Mock.On("Incr", mock.Anything, mock.AnythingOfType("string")).Return(int64(1), nil).Once()
		redisMock.Mock.On("Expire", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil).Once()
		redisMock.Mock.On("Get", mock.Anything, mock.AnythingOfType("string")).Return("112233", nil).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/verify-otp", bytes.NewReader(reqBody))
		r.Header.Set("Content-Type", "application/json")

		s.ServeHTTP(w, r)

		body := w.Body.String()
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
		assert.Contains(t, body, "invalid otp code")
	})
}

func TestLogin(t *testing.T) {
	setup(t)
	s.POST(constants.EndpointV1+"/auth/login", userHandler.Login)
	t.Run("Test 1 | Success Login", func(t *testing.T) {
		var err error
		userDataFromDB.Password, err = helpers.GenerateHash(userDataFromDB.Password)
		if err != nil {
			t.Error(err)
		}
		userDataFromDB.Active = true
		req := requests.UserLoginRequest{
			Email:    "patrick@gmail.com",
			Password: "Test123!@",
		}
		reqBody, _ := json.Marshal(req)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		jwtServiceMock.Mock.On("GenerateTokenPair", mock.AnythingOfType("string"), mock.AnythingOfType("bool"), mock.AnythingOfType("string")).Return(jwt.TokenPair{
			AccessToken:      "eyBlablablabla",
			RefreshToken:     "eyRefresh",
			AccessExpiresAt:  time.Now().Add(time.Hour),
			RefreshExpiresAt: time.Now().Add(24 * time.Hour),
			AccessJTI:        "access-jti",
			RefreshJTI:       "refresh-jti",
		}, nil).Once()
		redisMock.Mock.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Once()
		redisMock.Mock.On("Expire", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/login", bytes.NewReader(reqBody))
		r.Header.Set("Content-Type", "application/json")

		s.ServeHTTP(w, r)

		body := w.Body.String()
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Contains(t, body, "login success")
		assert.Contains(t, body, "ey")
	})
	t.Run("Test 2 | User is Not Exists", func(t *testing.T) {
		req := requests.UserLoginRequest{
			Email:    "patrick312@gmail.com",
			Password: "Test123!@",
		}
		reqBody, _ := json.Marshal(req)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(V1Domains.UserDomain{}, constants.ErrUserNotFound).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/login", bytes.NewReader(reqBody))
		r.Header.Set("Content-Type", "application/json")

		s.ServeHTTP(w, r)

		body := w.Body.String()
		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
		assert.Contains(t, body, "invalid email or password")
	})
}

func TestGetUserData(t *testing.T) {
	setup(t)
	s.GET("/users/me", userHandler.GetUserData)

	t.Run("Test 1 | Success Fetched User Data (cache miss)", func(t *testing.T) {
		ristrettoMock.Mock.On("Get", mock.AnythingOfType("string")).Return(nil).Once()
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		ristrettoMock.Mock.On("Set", mock.AnythingOfType("string"), mock.Anything).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		r.Header.Set("Content-Type", "application/json")

		s.ServeHTTP(w, r)

		body := w.Body.String()
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Contains(t, body, "user data fetched successfully")
	})

	t.Run("Test 2 | Failed to fetch User Data", func(t *testing.T) {
		ristrettoMock.Mock.On("Get", mock.AnythingOfType("string")).Return(nil).Once()
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(V1Domains.UserDomain{}, constants.ErrUnexpected).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		r.Header.Set("Content-Type", "application/json")

		s.ServeHTTP(w, r)

		assert.NotEqual(t, http.StatusOK, w.Result().StatusCode)
	})
}
