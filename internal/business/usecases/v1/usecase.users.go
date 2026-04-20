package v1

import (
	"context"
	"fmt"
	"time"

	V1Domains "github.com/snykk/go-rest-boilerplate/internal/business/domains/v1"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/mailer"
	"github.com/sirupsen/logrus"
)

// dummyBcryptHash is a pre-computed bcrypt hash used to mask the timing
// difference between "user not found" and "wrong password" branches in
// Login. Comparing an arbitrary password against it takes the same
// ~100ms a real bcrypt comparison does, preventing user enumeration.
// Generated from bcrypt.GenerateFromPassword([]byte("dummy"), DefaultCost).
const dummyBcryptHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

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

func (userUC *userUsecase) Store(ctx context.Context, inDom *V1Domains.UserDomain) (V1Domains.UserDomain, error) {
	hashed, err := helpers.GenerateHash(inDom.Password)
	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrInternal(fmt.Errorf("hash password: %w", err).Error())
	}
	inDom.Password = hashed

	inDom.CreatedAt = time.Now().In(constants.GMT7)
	if err := userUC.repo.Store(ctx, inDom); err != nil {
		return V1Domains.UserDomain{}, mapRepoError(err, "store user")
	}

	outDom, err := userUC.repo.GetByEmail(ctx, inDom)
	if err != nil {
		return V1Domains.UserDomain{}, mapRepoError(err, "fetch newly created user")
	}

	return outDom, nil
}

func (userUC *userUsecase) Login(ctx context.Context, inDom *V1Domains.UserDomain) (V1Domains.UserDomain, error) {
	userDomain, err := userUC.repo.GetByEmail(ctx, inDom)
	if err != nil {
		// Run a dummy bcrypt comparison so this path takes roughly the
		// same wall-clock time as a real password check. Without this,
		// an attacker can enumerate valid emails by measuring response
		// latency.
		_ = helpers.ValidateHash(inDom.Password, dummyBcryptHash)
		return V1Domains.UserDomain{}, constants.ErrUnauthorized("invalid email or password")
	}

	if !userDomain.Active {
		return V1Domains.UserDomain{}, constants.ErrForbidden("account is not activated")
	}

	if !helpers.ValidateHash(inDom.Password, userDomain.Password) {
		return V1Domains.UserDomain{}, constants.ErrUnauthorized("invalid email or password")
	}

	isAdmin := userDomain.RoleID == constants.AdminID
	token, err := userUC.jwtService.GenerateToken(userDomain.ID, isAdmin, userDomain.Email)
	if err != nil {
		return V1Domains.UserDomain{}, constants.ErrInternal(fmt.Errorf("generate token: %w", err).Error())
	}
	userDomain.Token = token

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
		return constants.ErrInternal(fmt.Errorf("generate otp: %w", err).Error())
	}

	if err = userUC.mailer.SendOTP(code, email); err != nil {
		logger.Error("failed to send OTP email", logrus.Fields{
			constants.LoggerCategory: constants.LoggerCategoryCache,
			"email":                  email,
			"error":                  err.Error(),
		})
		return constants.ErrInternal(fmt.Errorf("send otp: %w", err).Error())
	}

	// store OTP code in Redis and reset failed-attempt counter
	otpKey := fmt.Sprintf("user_otp:%s", email)
	if err = userUC.redisCache.Set(ctx, otpKey, code); err != nil {
		logger.Error("failed to cache OTP", logrus.Fields{
			constants.LoggerCategory: constants.LoggerCategoryCache,
			"email":                  email,
			"error":                  err.Error(),
		})
	}
	_ = userUC.redisCache.Del(ctx, otpAttemptsKey(email))

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

	// Brute-force guard: OTP is only 6 digits (1M combinations), so we
	// must hard-cap attempts per email. The counter lives in Redis with
	// the same TTL as the OTP itself.
	attemptsKey := otpAttemptsKey(email)
	attempts, err := userUC.redisCache.Incr(ctx, attemptsKey)
	if err != nil {
		logger.Error("failed to track OTP attempts", logrus.Fields{
			constants.LoggerCategory: constants.LoggerCategoryCache,
			"email":                  email,
			"error":                  err.Error(),
		})
	} else if attempts == 1 {
		// First attempt in this window — set expiry to match OTP TTL.
		ttl := time.Duration(config.AppConfig.REDISExpired) * time.Minute
		_ = userUC.redisCache.Expire(ctx, attemptsKey, ttl)
	}
	if attempts > int64(config.AppConfig.OTPMaxAttempts) {
		return constants.ErrForbidden("too many invalid otp attempts, please request a new code")
	}

	// retrieve OTP from Redis and validate
	otpKey := fmt.Sprintf("user_otp:%s", email)
	otpRedis, err := userUC.redisCache.Get(ctx, otpKey)
	if err != nil {
		return constants.ErrBadRequest("otp code expired or not found")
	}

	if otpRedis != userOTP {
		return constants.ErrBadRequest("invalid otp code")
	}

	// activate user
	if err = userUC.repo.ChangeActiveUser(ctx, &V1Domains.UserDomain{ID: domain.ID, Active: true}); err != nil {
		return constants.ErrInternal(fmt.Errorf("activate user: %w", err).Error())
	}

	// cleanup caches
	if err = userUC.redisCache.Del(ctx, otpKey); err != nil {
		logger.Error("failed to delete OTP cache", logrus.Fields{
			constants.LoggerCategory: constants.LoggerCategoryCache,
			"email":                  email,
			"error":                  err.Error(),
		})
	}
	_ = userUC.redisCache.Del(ctx, attemptsKey)
	userUC.ristrettoCache.Del("users", fmt.Sprintf("user/%s", email))

	return nil
}

func (userUC *userUsecase) GetByEmail(ctx context.Context, email string) (V1Domains.UserDomain, error) {
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

// otpAttemptsKey returns the Redis key that tracks failed VerifyOTP
// attempts for an email, scoped separately from the OTP code itself.
func otpAttemptsKey(email string) string {
	return fmt.Sprintf("otp_attempts:%s", email)
}

// mapRepoError preserves DomainError types returned from the repository
// while wrapping raw errors in a formatted internal error. Without this,
// errors.As(err, *DomainError) upstream would fail.
func mapRepoError(err error, op string) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*constants.DomainError); ok {
		return err
	}
	return constants.ErrInternal(fmt.Errorf("%s: %w", op, err).Error())
}
