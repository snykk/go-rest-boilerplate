package v1

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	V1Domains "github.com/snykk/go-rest-boilerplate/internal/business/domains/v1"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/mailer"
	"github.com/snykk/go-rest-boilerplate/pkg/observability"
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
		return V1Domains.UserDomain{}, apperror.InternalCause(fmt.Errorf("hash password: %w", err))
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
		return V1Domains.UserDomain{}, apperror.Unauthorized("invalid email or password")
	}

	if !userDomain.Active {
		return V1Domains.UserDomain{}, apperror.Forbidden("account is not activated")
	}

	if !helpers.ValidateHash(inDom.Password, userDomain.Password) {
		return V1Domains.UserDomain{}, apperror.Unauthorized("invalid email or password")
	}

	isAdmin := userDomain.RoleID == constants.AdminID
	pair, err := userUC.jwtService.GenerateTokenPair(userDomain.ID, isAdmin, userDomain.Email)
	if err != nil {
		return V1Domains.UserDomain{}, apperror.InternalCause(fmt.Errorf("generate token: %w", err))
	}
	if err := userUC.rememberRefresh(ctx, pair); err != nil {
		// If Redis is unavailable we'd rather fail login than issue a
		// refresh token the /refresh endpoint can't verify.
		return V1Domains.UserDomain{}, apperror.InternalCause(fmt.Errorf("persist refresh: %w", err))
	}
	userDomain.Token = pair.AccessToken
	userDomain.RefreshToken = pair.RefreshToken

	return userDomain, nil
}

// refreshKey scopes refresh-token jti entries so they don't collide
// with OTP keys in Redis.
func refreshKey(jti string) string { return fmt.Sprintf("refresh:%s", jti) }

// rememberRefresh stores the refresh jti in Redis with a TTL matching
// the refresh token's exp. /refresh and /logout treat absence here as
// "revoked", which is how logout works without an access-token
// blacklist.
func (userUC *userUsecase) rememberRefresh(ctx context.Context, pair jwt.TokenPair) error {
	ttl := time.Until(pair.RefreshExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("refresh token already expired")
	}
	if err := userUC.redisCache.Set(ctx, refreshKey(pair.RefreshJTI), pair.RefreshJTI); err != nil {
		return err
	}
	// Set() in this project applies the cache-wide expires in minutes;
	// override it explicitly so each refresh token has its own TTL.
	return userUC.redisCache.Expire(ctx, refreshKey(pair.RefreshJTI), ttl)
}

// Refresh verifies the supplied refresh token against the store,
// rotates it, and returns a new access+refresh pair. Replay of an
// already-used refresh token fails because rememberRefresh → Del
// makes the old jti unknown.
func (userUC *userUsecase) Refresh(ctx context.Context, refreshToken string) (V1Domains.UserDomain, error) {
	claims, err := userUC.jwtService.ParseRefreshToken(refreshToken)
	if err != nil {
		return V1Domains.UserDomain{}, apperror.Unauthorized("invalid refresh token")
	}

	// Verify the jti is still live server-side; logout / previous
	// rotation would have removed it.
	if _, err := userUC.redisCache.Get(ctx, refreshKey(claims.ID)); err != nil {
		return V1Domains.UserDomain{}, apperror.Unauthorized("refresh token has been revoked")
	}

	// Fresh identity lookup so revoked / deactivated accounts stop
	// getting new access tokens even while their refresh is live.
	userDomain, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: claims.Email})
	if err != nil {
		return V1Domains.UserDomain{}, apperror.Unauthorized("user no longer exists")
	}
	if !userDomain.Active {
		return V1Domains.UserDomain{}, apperror.Forbidden("account is not activated")
	}

	isAdmin := userDomain.RoleID == constants.AdminID
	pair, err := userUC.jwtService.GenerateTokenPair(userDomain.ID, isAdmin, userDomain.Email)
	if err != nil {
		return V1Domains.UserDomain{}, apperror.InternalCause(fmt.Errorf("generate token: %w", err))
	}

	// Rotate: remove the old jti, record the new one. Do this after
	// the new pair is minted so a mint failure doesn't leave the user
	// with no valid refresh token at all.
	if err := userUC.rememberRefresh(ctx, pair); err != nil {
		return V1Domains.UserDomain{}, apperror.InternalCause(fmt.Errorf("persist refresh: %w", err))
	}
	_ = userUC.redisCache.Del(ctx, refreshKey(claims.ID))

	userDomain.Token = pair.AccessToken
	userDomain.RefreshToken = pair.RefreshToken
	return userDomain, nil
}

// Logout revokes the refresh token so /refresh rejects it. Access
// tokens remain valid until their natural expiry — clients should
// discard them on logout. (A full blacklist would require checking
// every request against Redis; we trade that off for simplicity.)
func (userUC *userUsecase) Logout(ctx context.Context, refreshToken string) error {
	claims, err := userUC.jwtService.ParseRefreshToken(refreshToken)
	if err != nil {
		return apperror.Unauthorized("invalid refresh token")
	}
	if err := userUC.redisCache.Del(ctx, refreshKey(claims.ID)); err != nil {
		return apperror.InternalCause(fmt.Errorf("revoke refresh: %w", err))
	}
	return nil
}

func (userUC *userUsecase) SendOTP(ctx context.Context, email string) error {
	domain, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return apperror.NotFound("email not found")
	}

	if domain.Active {
		return apperror.BadRequest("account already activated")
	}

	code, err := helpers.GenerateOTPCode(6)
	if err != nil {
		return apperror.InternalCause(fmt.Errorf("generate otp: %w", err))
	}

	if err = userUC.mailer.SendOTP(code, email); err != nil {
		observability.ObserveMailerOp("queue_full")
		logger.Error("failed to enqueue OTP email", logrus.Fields{
			constants.LoggerCategory: constants.LoggerCategoryCache,
			"email":                  email,
			"error":                  err.Error(),
		})
		return apperror.InternalCause(fmt.Errorf("send otp: %w", err))
	}

	// store OTP code in Redis and reset failed-attempt counter
	otpKey := fmt.Sprintf("user_otp:%s", email)
	if err = userUC.redisCache.Set(ctx, otpKey, code); err != nil {
		observability.ObserveCacheOp("redis", "set", "error")
		logger.Error("failed to cache OTP", logrus.Fields{
			constants.LoggerCategory: constants.LoggerCategoryCache,
			"email":                  email,
			"error":                  err.Error(),
		})
	} else {
		observability.ObserveCacheOp("redis", "set", "ok")
	}
	_ = userUC.redisCache.Del(ctx, otpAttemptsKey(email))

	return nil
}

func (userUC *userUsecase) VerifyOTP(ctx context.Context, email string, userOTP string) error {
	domain, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return apperror.NotFound("email not found")
	}

	if domain.Active {
		return apperror.BadRequest("account already activated")
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
		return apperror.Forbidden("too many invalid otp attempts, please request a new code")
	}

	// retrieve OTP from Redis and validate
	otpKey := fmt.Sprintf("user_otp:%s", email)
	otpRedis, err := userUC.redisCache.Get(ctx, otpKey)
	if err != nil {
		observability.ObserveCacheOp("redis", "get", "miss")
		return apperror.BadRequest("otp code expired or not found")
	}
	observability.ObserveCacheOp("redis", "get", "hit")

	if otpRedis != userOTP {
		return apperror.BadRequest("invalid otp code")
	}

	// activate user
	if err = userUC.repo.ChangeActiveUser(ctx, &V1Domains.UserDomain{ID: domain.ID, Active: true}); err != nil {
		return apperror.InternalCause(fmt.Errorf("activate user: %w", err))
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
			observability.ObserveCacheOp("ristretto", "get", "hit")
			return cached, nil
		}
		observability.ObserveCacheOp("ristretto", "get", "error")
		logger.Info("cache type assertion failed, fetching from DB", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryCache})
	} else {
		observability.ObserveCacheOp("ristretto", "get", "miss")
	}

	user, err := userUC.repo.GetByEmail(ctx, &V1Domains.UserDomain{Email: email})
	if err != nil {
		return V1Domains.UserDomain{}, apperror.NotFound("email not found")
	}

	// populate cache
	userUC.ristrettoCache.Set(cacheKey, user)
	observability.ObserveCacheOp("ristretto", "set", "ok")

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
	if _, ok := err.(*apperror.DomainError); ok {
		return err
	}
	return apperror.InternalCause(fmt.Errorf("%s: %w", op, err))
}
