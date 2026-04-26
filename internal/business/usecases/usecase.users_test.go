package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/http/datatransfers/requests"
	"github.com/snykk/go-rest-boilerplate/internal/test/mocks"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	jwtServiceMock  *mocks.JWTService
	userRepoMock    *mocks.UserRepository
	mailerOTPMock   *mocks.OTPMailer
	redisMock       *mocks.RedisCache
	ristrettoMock   *mocks.RistrettoCache
	userUsecase     usecases.UserUsecase
	usersDataFromDB []entities.UserDomain
	userDataFromDB  entities.UserDomain
)

func setup(t *testing.T) {
	// VerifyOTP reads OTP_MAX_ATTEMPTS and REDIS_EXPIRED from config.
	// Seed non-zero defaults so tests don't trip the brute-force guard.
	config.AppConfig.OTPMaxAttempts = 5
	config.AppConfig.REDISExpired = 5

	mailerOTPMock = mocks.NewOTPMailer(t)
	jwtServiceMock = mocks.NewJWTService(t)
	userRepoMock = mocks.NewUserRepository(t)
	redisMock = mocks.NewRedisCache(t)
	ristrettoMock = mocks.NewRistrettoCache(t)
	userUsecase = usecases.NewUserUsecase(userRepoMock, jwtServiceMock, mailerOTPMock, redisMock, ristrettoMock, usecases.UserUsecaseConfig{OTPMaxAttempts: 5, OTPTTL: 5 * time.Minute})
	usersDataFromDB = []entities.UserDomain{
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

	userDataFromDB = entities.UserDomain{
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

		userRepoMock.Mock.On("Store", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(userDataFromDB, nil).Once()
		result, err := userUsecase.Store(context.Background(), req.ToV1Domain())

		assert.Nil(t, err)
		assert.NotEqual(t, "", result.ID)
		assert.Equal(t, 2, result.RoleID)
		assert.Equal(t, true, helpers.ValidateHash("11111", pass))
		assert.NotNil(t, result.CreatedAt)
	})

	t.Run("Test 2 | Failure When Store User Data", func(t *testing.T) {
		userRepoMock.Mock.On("Store", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(entities.UserDomain{}, constants.ErrUnexpected).Once()
		result, err := userUsecase.Store(context.Background(), req.ToV1Domain())

		assert.NotNil(t, err)
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

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(userDataFromDB, nil).Once()
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

		result, err := userUsecase.Login(context.Background(), req.ToV1Domain())

		assert.NotNil(t, result)
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

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(userDataFromDB, nil).Once()
		result, err := userUsecase.Login(context.Background(), req.ToV1Domain())

		assert.Equal(t, entities.UserDomain{}, result)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "account is not activated")
	})
	t.Run("Test 3 | Invalid Credential", func(t *testing.T) {
		req := requests.UserLoginRequest{
			Email:    "najibfikri13@gmail.com",
			Password: "111112",
		}
		userDataFromDB.Active = true
		userDataFromDB.Password, _ = helpers.GenerateHash(userDataFromDB.Password)

		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(userDataFromDB, nil).Once()

		result, err := userUsecase.Login(context.Background(), req.ToV1Domain())

		assert.Equal(t, entities.UserDomain{}, result)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "invalid email or password")
	})
}

func TestSendOTP(t *testing.T) {
	setup(t)
	t.Run("Test 1 | Success Send OTP", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(userDataFromDB, nil).Once()
		mailerOTPMock.On("SendOTP", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Once()
		redisMock.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil).Once()
		redisMock.On("Del", mock.Anything, mock.AnythingOfType("string")).Return(nil).Once()

		err := userUsecase.SendOTP(context.Background(), "najibfikri13@gmail.com")

		assert.Nil(t, err)
	})

	t.Run("Test 2 | Email Not Registered", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(entities.UserDomain{}, constants.ErrUserNotFound).Once()

		err := userUsecase.SendOTP(context.Background(), "najibfikri13@gmail.com")

		assert.NotNil(t, err)
	})
	t.Run("Test 3 | Failure When Send OTP", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(userDataFromDB, nil).Once()
		mailerOTPMock.On("SendOTP", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(constants.ErrUnexpected).Once()

		err := userUsecase.SendOTP(context.Background(), "najibfikri13@gmail.com")

		assert.NotNil(t, err)
	})
}

func TestVerifyOTP(t *testing.T) {
	setup(t)
	t.Run("Test 1 | Success Verify OTP", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(userDataFromDB, nil).Once()
		redisMock.Mock.On("Incr", mock.Anything, mock.AnythingOfType("string")).Return(int64(1), nil).Once()
		redisMock.Mock.On("Expire", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil).Once()
		redisMock.Mock.On("Get", mock.Anything, mock.AnythingOfType("string")).Return("112233", nil).Once()
		userRepoMock.Mock.On("ChangeActiveUser", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(nil).Once()
		redisMock.On("Del", mock.Anything, mock.AnythingOfType("string")).Return(nil).Twice()
		ristrettoMock.On("Del", "users", mock.AnythingOfType("string")).Once()

		err := userUsecase.VerifyOTP(context.Background(), "najibfikri13@gmail.com", "112233")

		assert.Nil(t, err)
	})
	t.Run("Test 2 | Email Not Registered", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(entities.UserDomain{}, constants.ErrUserNotFound).Once()

		err := userUsecase.VerifyOTP(context.Background(), "najibfikri13@gmail.com", "112233")

		assert.NotNil(t, err)
	})
	t.Run("Test 3 | Account Already Activated", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(usersDataFromDB[0], nil).Once()

		err := userUsecase.VerifyOTP(context.Background(), "najibfikri13@gmail.com", "112233")

		assert.NotNil(t, err)
	})
	t.Run("Test 4 | Invalid OTP Code", func(t *testing.T) {
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(userDataFromDB, nil).Once()
		redisMock.Mock.On("Incr", mock.Anything, mock.AnythingOfType("string")).Return(int64(1), nil).Once()
		redisMock.Mock.On("Expire", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil).Once()
		redisMock.Mock.On("Get", mock.Anything, mock.AnythingOfType("string")).Return("112233", nil).Once()

		err := userUsecase.VerifyOTP(context.Background(), "najibfikri13@gmail.com", "999999")

		assert.NotNil(t, err)
	})
}

func TestGetByEmail(t *testing.T) {
	setup(t)
	t.Run("Test 1 | Success Get User Data By Email (cache miss)", func(t *testing.T) {
		ristrettoMock.Mock.On("Get", mock.AnythingOfType("string")).Return(nil).Once()
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(userDataFromDB, nil).Once()
		ristrettoMock.Mock.On("Set", mock.AnythingOfType("string"), mock.Anything).Once()

		result, err := userUsecase.GetByEmail(context.Background(), "najibfikri13@gmail.com")

		assert.Nil(t, err)
		assert.Equal(t, userDataFromDB, result)
	})

	t.Run("Test 2 | Success Get User Data By Email (cache hit)", func(t *testing.T) {
		ristrettoMock.Mock.On("Get", mock.AnythingOfType("string")).Return(userDataFromDB).Once()

		result, err := userUsecase.GetByEmail(context.Background(), "najibfikri13@gmail.com")

		assert.Nil(t, err)
		assert.Equal(t, userDataFromDB, result)
	})

	t.Run("Test 3 | User doesn't exist", func(t *testing.T) {
		ristrettoMock.Mock.On("Get", mock.AnythingOfType("string")).Return(nil).Once()
		userRepoMock.Mock.On("GetByEmail", mock.Anything, mock.AnythingOfType("*entities.UserDomain")).Return(entities.UserDomain{}, apperror.NotFound("email not found")).Once()

		result, err := userUsecase.GetByEmail(context.Background(), "johndoe@gmail.com")

		assert.Equal(t, entities.UserDomain{}, result)
		assert.NotNil(t, err)
	})
}
