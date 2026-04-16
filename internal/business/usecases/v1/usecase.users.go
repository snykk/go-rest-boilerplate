package v1

import (
	"context"
	"fmt"
	"time"

	V1Domains "github.com/snykk/go-rest-boilerplate/internal/business/domains/v1"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/mailer"
	"github.com/sirupsen/logrus"
)

type userUsecase struct {
	jwtService     jwt.JWTService
	repo           V1Domains.UserRepository
	mailer         mailer.OTPMailer
	redisCache     caches.RedisCache
	ristrettoCache caches.RistrettoCache
}

func NewUserUsecase(repo V1Domains.UserRepository, jwtService jwt.JWTService, mailer mailer.OTPMailer, redisCache caches.RedisCache, ristrettoCache caches.RistrettoCache) V1Domains.UserUsecase {
	return &userUsecase{
		repo:           repo,
		jwtService:     jwtService,
		mailer:         mailer,
		redisCache:     redisCache,
		ristrettoCache: ristrettoCache,
	}
}

func (userUC *userUsecase) Store(ctx context.Context, inDom *V1Domains.UserDomain) (outDom V1Domains.UserDomain, err error) {
	inDom.Password, err = helpers.GenerateHash(inDom.Password)
	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrInternal(err.Error())
	}

	inDom.CreatedAt = time.Now().In(constants.GMT7)
	err = userUC.repo.Store(ctx, inDom)
	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrInternal(err.Error())
	}

	outDom, err = userUC.repo.GetByEmail(ctx, inDom)
	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrInternal(err.Error())
	}

	return outDom, nil
}

func (userUC *userUsecase) Login(ctx context.Context, inDom *V1Domains.UserDomain) (outDom V1Domains.UserDomain, err error) {
	userDomain, err := userUC.repo.GetByEmail(ctx, inDom)
	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrUnauthorized("invalid email or password")
	}

	if !userDomain.Active {
		return V1Domains.UserDomain{}, constants.ErrForbidden("account is not activated")
	}

	if !helpers.ValidateHash(inDom.Password, userDomain.Password) {
		return V1Domains.UserDomain{}, constants.ErrUnauthorized("invalid email or password")
	}

	if userDomain.RoleID == constants.AdminID {
		userDomain.Token, err = userUC.jwtService.GenerateToken(userDomain.ID, true, userDomain.Email)
	} else {
		userDomain.Token, err = userUC.jwtService.GenerateToken(userDomain.ID, false, userDomain.Email)
	}

	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrInternal(err.Error())
	}

	return userDomain, nil
}

func (userUC *userUsecase) SendOTP(ctx context.Context, email string) error {
	domain, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return constants.ErrNotFound("email not found")
	}

	if domain.Active {
		return constants.ErrBadRequest("account already activated")
	}

	code, err := helpers.GenerateOTPCode(6)
	if err != nil {
		return constants.ErrInternal(err.Error())
	}

	if err = userUC.mailer.SendOTP(code, email); err != nil {
		return constants.ErrInternal(err.Error())
	}

	// store OTP code in Redis
	otpKey := fmt.Sprintf("user_otp:%s", email)
	if err = userUC.redisCache.Set(otpKey, code); err != nil {
		logger.InfoF("failed to cache OTP: %v", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryCache}, err)
	}

	return nil
}

func (userUC *userUsecase) VerifyOTP(ctx context.Context, email string, userOTP string) error {
	domain, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return constants.ErrNotFound("email not found")
	}

	if domain.Active {
		return constants.ErrBadRequest("account already activated")
	}

	// retrieve OTP from Redis and validate
	otpKey := fmt.Sprintf("user_otp:%s", email)
	otpRedis, err := userUC.redisCache.Get(otpKey)
	if err != nil {
		return constants.ErrInternal("otp code expired or not found")
	}

	if otpRedis != userOTP {
		return constants.ErrBadRequest("invalid otp code")
	}

	// activate user
	if err = userUC.repo.ChangeActiveUser(ctx, &V1Domains.UserDomain{ID: domain.ID, Active: true}); err != nil {
		return constants.ErrInternal(err.Error())
	}

	// cleanup caches
	if err = userUC.redisCache.Del(otpKey); err != nil {
		logger.InfoF("failed to delete OTP cache: %v", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryCache}, err)
	}
	userUC.ristrettoCache.Del("users", fmt.Sprintf("user/%s", email))

	return nil
}

func (userUC *userUsecase) GetByEmail(ctx context.Context, email string) (outDom V1Domains.UserDomain, err error) {
	// check in-memory cache first
	cacheKey := fmt.Sprintf("user/%s", email)
	if val := userUC.ristrettoCache.Get(cacheKey); val != nil {
		if cached, ok := val.(V1Domains.UserDomain); ok {
			return cached, nil
		}
		logger.Info("cache type assertion failed, fetching from DB", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryCache})
	}

	user, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrNotFound("email not found")
	}

	// populate cache
	userUC.ristrettoCache.Set(cacheKey, user)

	return user, nil
}
