package v1_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	V1Domains "github.com/snykk/go-rest-boilerplate/internal/business/domains/v1"
	V1Usecases "github.com/snykk/go-rest-boilerplate/internal/business/usecases/v1"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	"github.com/snykk/go-rest-boilerplate/internal/mocks"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	jwtServiceMock  *mocks.JWTService
	userRepoMock    *mocks.UserRepository
	mailerOTPMock   *mocks.OTPMailer
	userUsecase     V1Domains.UserUsecase
	usersDataFromDB []V1Domains.UserDomain
	userDataFromDB  V1Domains.UserDomain
)

func setup(t *testing.T) {
	mailerOTPMock = mocks.NewOTPMailer(t)
	jwtServiceMock = mocks.NewJWTService(t)
	userRepoMock = mocks.NewUserRepository(t)
	userUsecase = V1Usecases.NewUserUsecase(userRepoMock, jwtServiceMock, mailerOTPMock)
	usersDataFromDB = []V1Domains.UserDomain{
		{
			ID:        "ddfcea5c-d919-4a8f-a631-4ace39337s3a",
			Username:  "itsmepatrick",
			Email:     "najibfikri13@gmail.com",
			RoleID:    1,
			Password:  "11111",
			Active:    true,
			CreatedAt: time.Now(),
		},
		{
			ID:        "wifff3jd-idhd-0sis-8dua-4fiefie37kfj",
			Username:  "johny",
			Email:     "johny123@gmail.com",
			RoleID:    2,
			Password:  "11111",
			Active:    true,
			CreatedAt: time.Now(),
		},
	}

	userDataFromDB = V1Domains.UserDomain{
		ID:        "fjskeie8-jfk8-qke0-sksj-ksjf89e8ehfu",
		Username:  "itsmepatrick",
		Email:     "najibfikri13@gmail.com",
		Password:  "11111",
		RoleID:    2,
		Active:    false,
		CreatedAt: time.Now(),
	}
}

func TestStore(t *testing.T) {
	setup(t)
	req := requests.UserRequest{
		Username: "itsmepatrick",
		Email:    "najibfikri13@gmail.com",
		Password: "11111",
	}
	t.Run("Test 1 | Success Store User Data", func(t *testing.T) {
		pass, _ := helpers.GenerateHash("11111")

		userRepoMock.Mock.On("Store", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(nil).Once()
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		result, statusCode, err := userUsecase.Store(context.Background(), req.ToV1Domain())

		assert.Nil(t, err)
		assert.Equal(t, http.StatusCreated, statusCode)
		assert.NotEqual(t, "", result.ID)
		assert.Equal(t, 2, result.RoleID)
		assert.Equal(t, true, helpers.ValidateHash("11111", pass))
		assert.NotNil(t, result.CreatedAt)
	})

	t.Run("Test 2 | Failure When Store User Data", func(t *testing.T) {
		userRepoMock.Mock.On("Store", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(constants.ErrUnexpected).Once()
		result, statusCode, err := userUsecase.Store(context.Background(), req.ToV1Domain())

		assert.NotNil(t, err)
		assert.Equal(t, http.StatusInternalServerError, statusCode)
		assert.Equal(t, "", result.ID)
	})

}

func TestLogin(t *testing.T) {
	setup(t)
	t.Run("Test 1 | Success Login", func(t *testing.T) {
		req := requests.UserLoginRequest{
			Email:    "najibfikri13@gmail.com",
			Password: "11111",
		}
		userDataFromDB.Active = true
		userDataFromDB.Password, _ = helpers.GenerateHash(userDataFromDB.Password)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		jwtServiceMock.Mock.On("GenerateToken", mock.AnythingOfType("string"), mock.AnythingOfType("bool"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("eyBlablablabla", nil).Once()

		result, statusCode, err := userUsecase.Login(context.Background(), req.ToV1Domain())

		assert.NotNil(t, result)
		assert.Equal(t, http.StatusOK, statusCode)
		assert.Nil(t, err)
		assert.Contains(t, result.Token, "ey")
	})
	t.Run("Test 2 | Account Not Activated Yet", func(t *testing.T) {
		req := requests.UserLoginRequest{
			Email:    "najibfikri13@gmail.com",
			Password: "11111",
		}
		userDataFromDB.Active = false
		userDataFromDB.Password, _ = helpers.GenerateHash(userDataFromDB.Password)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		result, statusCode, err := userUsecase.Login(context.Background(), req.ToV1Domain())

		assert.Equal(t, V1Domains.UserDomain{}, result)
		assert.Equal(t, http.StatusForbidden, statusCode)
		assert.NotNil(t, err)
		assert.Equal(t, errors.New("account is not activated"), err)
		assert.Equal(t, "", result.Token)
	})
	t.Run("Test 3 | Invalid Credential", func(t *testing.T) {
		req := requests.UserLoginRequest{
			Email:    "najibfikri13@gmail.com",
			Password: "111112",
		}
		userDataFromDB.Active = true
		userDataFromDB.Password, _ = helpers.GenerateHash(userDataFromDB.Password)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()

		result, statusCode, err := userUsecase.Login(context.Background(), req.ToV1Domain())

		assert.Equal(t, V1Domains.UserDomain{}, result)
		assert.NotNil(t, err)
		assert.Equal(t, http.StatusUnauthorized, statusCode)
		assert.Equal(t, errors.New("invalid email or password"), err)
		assert.Equal(t, "", result.Token)
	})
}

func TestActivate(t *testing.T) {
	setup(t)
	t.Run("Test 1 | Success Activate Email", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		userRepoMock.Mock.On("ChangeActiveUser", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(nil).Once()

		statusCode, err := userUsecase.ActivateUser(context.Background(), "najibfikri13@gmail.com")

		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, statusCode)
	})

	t.Run("Test 2 | Failure When Activate Email", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(V1Domains.UserDomain{}, errors.New("email not found")).Once()

		statusCode, err := userUsecase.ActivateUser(context.Background(), "johndoe@gmail.com")

		assert.NotNil(t, err)
		assert.Equal(t, http.StatusNotFound, statusCode)
	})
}

func TestSendOTP(t *testing.T) {
	setup(t)
	t.Run("Test 1 | Success Send OTP", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		mailerOTPMock.On("SendOTP", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Once()

		otpCode, statusCode, err := userUsecase.SendOTP(context.Background(), "najibfikri13@gmail.com")

		assert.Nil(t, err)
		assert.NotEqual(t, "", otpCode)
		assert.Equal(t, http.StatusOK, statusCode)
	})

	t.Run("Test 2 | Email Not Registered", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(V1Domains.UserDomain{}, constants.ErrUserNotFound).Once()

		otpCode, statusCode, err := userUsecase.SendOTP(context.Background(), "najibfikri13@gmail.com")

		assert.NotNil(t, err)
		assert.Equal(t, "", otpCode)
		assert.Equal(t, http.StatusNotFound, statusCode)
	})
	t.Run("Test 3 | Failure When Send OTP", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()
		mailerOTPMock.On("SendOTP", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(constants.ErrUnexpected).Once()

		otpCode, statusCode, err := userUsecase.SendOTP(context.Background(), "najibfikri13@gmail.com")

		assert.NotNil(t, err)
		assert.Equal(t, "", otpCode)
		assert.Equal(t, http.StatusInternalServerError, statusCode)
	})
}

func TestVerifOTP(t *testing.T) {
	setup(t)
	t.Run("Test 1 | Success Verify OTP", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()

		statusCode, err := userUsecase.VerifOTP(context.Background(), "najibfikri13@gmail.com", "112233", "112233")

		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, statusCode)
	})
	t.Run("Test 2 | Email Not Registered", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(V1Domains.UserDomain{}, constants.ErrUserNotFound).Once()

		statusCode, err := userUsecase.VerifOTP(context.Background(), "najibfikri13@gmail.com", "112233", "112233")

		assert.NotNil(t, err)
		assert.Equal(t, http.StatusNotFound, statusCode)
	})
	t.Run("Test 3 | Account Already Activated", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(usersDataFromDB[0], nil).Once()

		statusCode, err := userUsecase.VerifOTP(context.Background(), "najibfikri13@gmail.com", "112233", "112233")

		assert.NotNil(t, err)
		assert.Equal(t, http.StatusBadRequest, statusCode)
	})
	t.Run("Test 4 | Invalid OTP Code", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*v1.UserDomain")).Return(userDataFromDB, nil).Once()

		statusCode, err := userUsecase.VerifOTP(context.Background(), "najibfikri13@gmail.com", "999999", "112233")

		assert.NotNil(t, err)
		assert.Equal(t, http.StatusBadRequest, statusCode)
	})
}
