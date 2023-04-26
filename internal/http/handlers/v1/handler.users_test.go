package v1_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dgriJWT "github.com/dgrijalva/jwt-go"
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
	usersDataFromDB []V1Domains.UserDomain
	userDataFromDB  V1Domains.UserDomain
	redisMock       *mocks.RedisCache
	ristrettoMock   *mocks.RistrettoCache
	s               *gin.Engine
)

func setup(t *testing.T) {
	jwtServiceMock = mocks.NewJWTService(t)
	redisMock = mocks.NewRedisCache(t)
	mailerOTPMock = mocks.NewOTPMailer(t)
	ristrettoMock = mocks.NewRistrettoCache(t)
	userRepoMock = mocks.NewUserRepository(t)
	userUsecase = V1Usecases.NewUserUsecase(userRepoMock, jwtServiceMock, mailerOTPMock)
	userHandler = V1Handlers.NewUserHandler(userUsecase, redisMock, ristrettoMock)

	usersDataFromDB = []V1Domains.UserDomain{
		{
			ID:        "ddfcea5c-d919-4a8f-a631-4ace39337s3a",
			Username:  "itsmepatrick",
			Email:     "najibfikri13@gmail.com",
			RoleID:    1,
			Password:  "23123sdf!",
			Active:    true,
			CreatedAt: time.Now(),
		},
		{
			ID:        "wifff3jd-idhd-0sis-8dua-4fiefie37kfj",
			Username:  "johny",
			Email:     "johny123@gmail.com",
			RoleID:    2,
			Password:  "23123sdf!",
			Active:    true,
			CreatedAt: time.Now(),
		},
	}

	userDataFromDB = V1Domains.UserDomain{
		ID:        "fjskeie8-jfk8-qke0-sksj-ksjf89e8ehfu",
		Username:  "itsmepatrick",
		Email:     "najibfikri13@gmail.com",
		Password:  "23123sdf!",
		RoleID:    2,
		Active:    false,
		CreatedAt: time.Now(),
	}

	// Create gin engine
	s = gin.Default()
	s.Use(lazyAuth)
}

func lazyAuth(ctx *gin.Context) {
	// hash
	pass, _ := helpers.GenerateHash(userDataFromDB.Password)
	// prepare claims
	jwtClaims := jwt.JwtCustomClaim{
		UserID:   userDataFromDB.ID,
		IsAdmin:  false,
		Email:    userDataFromDB.Email,
		Password: pass,
		StandardClaims: dgriJWT.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * time.Duration(config.AppConfig.JWTExpired)).Unix(),
			Issuer:    userDataFromDB.Username,
			IssuedAt:  time.Now().Unix(),
		},
	}
	ctx.Set(constants.CtxAuthenticatedUserKey, jwtClaims)
}

func TestRegis(t *testing.T) {
	setup(t)
	// Define route
	s.POST(constants.EndpointV1+"/auth/regis", userHandler.Regis)
	t.Run("When Success Regis", func(t *testing.T) {
		req := requests.UserRequest{
			Username: "itsmepatrick",
			Email:    "najibfikri13@gmail.com",
			Password: "23123sdf!",
		}
		reqBody, _ := json.Marshal(req)

		userRepoMock.Mock.On("Store", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(nil).Once()
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/regis", bytes.NewReader(reqBody))

		r.Header.Set("Content-Type", "application/json")

		// Perform request
		s.ServeHTTP(w, r)

		body := w.Body.String()

		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusCreated, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
		assert.Contains(t, body, "registration user success")
	})
	t.Run("When Failure", func(t *testing.T) {
		t.Run("When Request is Empty", func(t *testing.T) {
			req := requests.UserRequest{}
			reqBody, _ := json.Marshal(req)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/regis", bytes.NewReader(reqBody))

			r.Header.Set("Content-Type", "application/json")

			// Perform request
			s.ServeHTTP(w, r)

			body := w.Body.String()

			// Assertions
			// Assert status code
			assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
			assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
			assert.Contains(t, body, "required")
		})
	})
}

func TestSendOTP(t *testing.T) {
	setup(t)
	// Define route
	s.POST(constants.EndpointV1+"/auth/send-otp", userHandler.SendOTP)
	t.Run("Test 1 | Success Send OTP", func(t *testing.T) {
		req := requests.UserSendOTPRequest{
			Email: "najibfikri13@gmail.com",
		}
		reqBody, _ := json.Marshal(req)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		mailerOTPMock.On("SendOTP", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Once()
		redisMock.On("Set", mock.AnythingOfType("string"), mock.Anything).Return(nil).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/send-otp", bytes.NewReader(reqBody))

		r.Header.Set("Content-Type", "application/json")

		// Perform request
		s.ServeHTTP(w, r)

		body := w.Body.String()

		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
		assert.Contains(t, body, "otp code has been send")
	})
	t.Run("Test 3 | Payloads is Empty", func(t *testing.T) {
		req := requests.UserSendOTPRequest{}
		reqBody, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/send-otp", bytes.NewReader(reqBody))

		r.Header.Set("Content-Type", "application/json")

		// Perform request
		s.ServeHTTP(w, r)

		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
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

		// Perform request
		s.ServeHTTP(w, r)

		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
	})
}

func TestVerifOTP(t *testing.T) {
	setup(t)
	// Define route
	s.POST(constants.EndpointV1+"/auth/verif-otp", userHandler.VerifOTP)
	t.Run("Test 1 | Success Verify OTP", func(t *testing.T) {
		req := requests.UserVerifOTPRequest{
			Email: "najibfikri13@gmail.com",
			Code:  "112233",
		}
		reqBody, _ := json.Marshal(req)

		redisMock.Mock.On("Get", mock.AnythingOfType("string")).Return("112233", nil)
		redisMock.On("Del", mock.AnythingOfType("string")).Return(nil).Once()
		ristrettoMock.On("Del", mock.AnythingOfType("string")).Once()

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		userRepoMock.Mock.On("ChangeActiveUser", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(nil).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/verif-otp", bytes.NewReader(reqBody))

		r.Header.Set("Content-Type", "application/json")

		// Perform request
		s.ServeHTTP(w, r)

		body := w.Body.String()

		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
		assert.Contains(t, body, "otp verification success")
	})
	t.Run("Test 2 | Payloads is Empty", func(t *testing.T) {
		req := requests.UserVerifOTPRequest{}
		reqBody, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/verif-otp", bytes.NewReader(reqBody))

		r.Header.Set("Content-Type", "application/json")

		// Perform request
		s.ServeHTTP(w, r)

		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
	})
	t.Run("Test 1 | Invalid OTP Code", func(t *testing.T) {
		req := requests.UserVerifOTPRequest{
			Email: "najibfikri13@gmail.com",
			Code:  "999999",
		}
		reqBody, _ := json.Marshal(req)

		redisMock.Mock.On("Get", mock.AnythingOfType("string")).Return("112233", nil)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/verif-otp", bytes.NewReader(reqBody))

		r.Header.Set("Content-Type", "application/json")

		// Perform request
		s.ServeHTTP(w, r)

		body := w.Body.String()

		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
		assert.Contains(t, body, "invalid otp code")
	})
}

func TestLogin(t *testing.T) {
	setup(t)
	// Define route
	s.POST(constants.EndpointV1+"/auth/login", userHandler.Login)
	t.Run("Test 1 | Success Login", func(t *testing.T) {
		// hash password field
		var err error
		userDataFromDB.Password, err = helpers.GenerateHash(userDataFromDB.Password)
		if err != nil {
			t.Error(err)
		}
		// make account activated
		userDataFromDB.Active = true
		req := requests.UserLoginRequest{
			Email:    "patrick@gmail.com",
			Password: "23123sdf!",
		}
		reqBody, _ := json.Marshal(req)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		jwtServiceMock.Mock.On("GenerateToken", mock.AnythingOfType("string"), mock.AnythingOfType("bool"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("eyBlablablabla", nil).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/login", bytes.NewReader(reqBody))

		r.Header.Set("Content-Type", "application/json")

		// Perform request
		s.ServeHTTP(w, r)

		body := w.Body.String()

		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
		assert.Contains(t, body, "login success")
		assert.Contains(t, body, "ey")
	})
	t.Run("Test 2 | User is Not Exists", func(t *testing.T) {
		req := requests.UserLoginRequest{
			Email:    "patrick312@gmail.com",
			Password: "23123sdf!",
		}
		reqBody, _ := json.Marshal(req)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(V1Domains.UserDomain{}, constants.ErrUserNotFound).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, constants.EndpointV1+"/auth/login", bytes.NewReader(reqBody))

		r.Header.Set("Content-Type", "application/json")

		// Perform request
		s.ServeHTTP(w, r)

		body := w.Body.String()

		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
		assert.Contains(t, body, "invalid email or password")
	})
}

func TestGetUserData(t *testing.T) {
	setup(t)
	// Define route
	s.GET("/users/me", userHandler.GetUserData)

	authenticatedUserEmail := userDataFromDB.Email
	t.Run("Test 1 | Success Fetched User Data", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		ristrettoMock.Mock.On("Get", fmt.Sprintf("user/%s", authenticatedUserEmail)).Return(nil).Once()
		ristrettoMock.Mock.On("Set", fmt.Sprintf("user/%s", authenticatedUserEmail), mock.Anything).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/users/me", nil)

		r.Header.Set("Content-Type", "application/json")

		// Perform request
		s.ServeHTTP(w, r)

		// parsing json to raw text
		body := w.Body.String()

		// Assertions
		// Assert status code
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
		assert.Contains(t, body, "user data fetched successfully")
	})

	t.Run("Test 2 | Failed to fetch User Data", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(V1Domains.UserDomain{}, constants.ErrUnexpected).Once()
		ristrettoMock.Mock.On("Get", fmt.Sprintf("user/%s", authenticatedUserEmail)).Return(nil).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/users/me", nil)

		r.Header.Set("Content-Type", "application/json")

		// Perform request
		s.ServeHTTP(w, r)

		// Assertions
		// Assert status code
		assert.NotEqual(t, http.StatusOK, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
	})
}
