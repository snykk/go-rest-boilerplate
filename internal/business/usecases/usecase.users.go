// Package usecases is the application business-logic layer. It owns
// the input ports (UserUsecase) and the gateway ports (UserRepository,
// caches, mailers) — interfaces describing what the use case needs
// from the outside world. Implementations live in
// internal/datasources/* and pkg/*; nothing in this package imports
// those concrete adapters.
package usecases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
	"github.com/snykk/go-rest-boilerplate/internal/config"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/mailer"
	"github.com/snykk/go-rest-boilerplate/pkg/observability"
	"golang.org/x/sync/singleflight"
)

// ───────────────────────────── ports ─────────────────────────────

// UserUsecase is the input boundary the HTTP handler talks to.
type UserUsecase interface {
	// Store hashes the password, creates the user row, and returns the
	// persisted record. New accounts start with active=false until OTP
	// verification flips them on.
	Store(ctx context.Context, inDom *entities.UserDomain) (outDom entities.UserDomain, err error)
	// Login validates credentials and returns a fresh access+refresh
	// token pair. Wrong password and unknown email take the same wall
	// time to mask user enumeration.
	Login(ctx context.Context, inDom *entities.UserDomain) (outDom entities.UserDomain, err error)
	// SendOTP generates a 6-digit code, stores it in Redis with TTL,
	// and enqueues the email via the async mailer. The HTTP response
	// returns on enqueue, not on actual SMTP delivery.
	SendOTP(ctx context.Context, email string) error
	// VerifyOTP checks the supplied code against Redis, increments a
	// per-email attempt counter, and activates the account on success.
	// Lockout fires after OTP_MAX_ATTEMPTS failures.
	VerifyOTP(ctx context.Context, email string, userOTP string) error
	// GetByEmail returns the user, hitting the in-memory cache first
	// and coalescing concurrent misses through singleflight.
	GetByEmail(ctx context.Context, email string) (outDom entities.UserDomain, err error)
	// Refresh verifies and rotates the refresh token, mints a new
	// access+refresh pair, and revokes the old jti.
	Refresh(ctx context.Context, refreshToken string) (outDom entities.UserDomain, err error)
	// Logout revokes the supplied refresh token by deleting its jti
	// from Redis. Access tokens remain valid until their natural exp.
	Logout(ctx context.Context, refreshToken string) error
}

// ListFilter narrows down List() results. Each field is optional;
// the empty value means "no filter on this dimension".
type ListFilter struct {
	RoleID         int  // 0 = any role
	ActiveOnly     bool // true = only active=true users
	IncludeDeleted bool // false (default) = WHERE deleted_at IS NULL
}

// UserRepository is the gateway port the use case needs to load and
// persist users. The interface lives here, in the use case package,
// because the use case dictates what it needs from the outside —
// implementations adapt to it (internal/datasources/repositories/...).
type UserRepository interface {
	// Store inserts the user and returns the persisted row in a single
	// round-trip so callers don't need a follow-up GetByEmail (which
	// would orphan the INSERT if it failed). Duplicate username/email
	// surfaces as apperror.Conflict.
	Store(ctx context.Context, inDom *entities.UserDomain) (entities.UserDomain, error)
	// GetByEmail looks up a user by email, excluding soft-deleted
	// rows. Returns apperror.NotFound when no row matches.
	GetByEmail(ctx context.Context, inDom *entities.UserDomain) (outDomain entities.UserDomain, err error)
	// GetByID looks up a user by primary key, excluding soft-deleted
	// rows. Returns apperror.NotFound when no row matches.
	GetByID(ctx context.Context, id string) (entities.UserDomain, error)
	// List returns users matching filter, paginated by offset/limit.
	// Limit is hard-capped server-side so a misbehaving caller can't
	// pull the whole table.
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]entities.UserDomain, error)
	// ChangeActiveUser flips the active flag (used by the OTP-verify
	// flow) and stamps updated_at. No-op on soft-deleted rows.
	ChangeActiveUser(ctx context.Context, inDom *entities.UserDomain) (err error)
	// SoftDelete sets deleted_at = NOW() so the row stays in the table
	// for audit/restore but stops matching default queries. Returns
	// apperror.NotFound if the row doesn't exist or is already deleted.
	SoftDelete(ctx context.Context, id string) error
}

// ─────────────────────── helpers / constants ───────────────────────

// normalizeEmail trims whitespace and lowercases the address so
// "User@Example.com " and "user@example.com" hash to the same Redis
// key, query the same DB row, and produce the same uniqueness
// violation. RFC 5321 says the local part is technically
// case-sensitive, but every consumer-grade mail provider treats it
// case-insensitively; matching that expectation avoids "I can't log in
// because I capitalized the U" support tickets.
func normalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// dummyBcryptHash is a pre-computed bcrypt hash used to mask the timing
// difference between "user not found" and "wrong password" branches in
// Login. Comparing an arbitrary password against it takes the same
// ~100ms a real bcrypt comparison does, preventing user enumeration.
// Generated from bcrypt.GenerateFromPassword([]byte("dummy"), DefaultCost).
const dummyBcryptHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// ─────────────────────── implementation ───────────────────────

type userUsecase struct {
	jwtService     jwt.JWTService
	repo           UserRepository
	mailer         mailer.OTPMailer
	redisCache     caches.RedisCache
	ristrettoCache caches.RistrettoCache

	// userByEmailGroup coalesces concurrent cache misses for the
	// same email so a thundering herd can't fan out into N parallel
	// DB round-trips. The group is keyed by normalized email.
	userByEmailGroup singleflight.Group
}

func NewUserUsecase(repo UserRepository, jwtService jwt.JWTService, mailer mailer.OTPMailer, redisCache caches.RedisCache, ristrettoCache caches.RistrettoCache) UserUsecase {
	return &userUsecase{
		repo:           repo,
		jwtService:     jwtService,
		mailer:         mailer,
		redisCache:     redisCache,
		ristrettoCache: ristrettoCache,
	}
}

func (userUC *userUsecase) Store(ctx context.Context, inDom *entities.UserDomain) (entities.UserDomain, error) {
	hashed, err := helpers.GenerateHash(inDom.Password)
	if err != nil {
		return entities.UserDomain{}, apperror.InternalCause(fmt.Errorf("hash password: %w", err))
	}
	inDom.Password = hashed
	inDom.Email = normalizeEmail(inDom.Email)
	inDom.CreatedAt = time.Now().In(constants.GMT7)

	stored, err := userUC.repo.Store(ctx, inDom)
	if err != nil {
		return entities.UserDomain{}, mapRepoError(err, "store user")
	}
	return stored, nil
}

func (userUC *userUsecase) Login(ctx context.Context, inDom *entities.UserDomain) (entities.UserDomain, error) {
	inDom.Email = normalizeEmail(inDom.Email)
	userDomain, err := userUC.repo.GetByEmail(ctx, inDom)
	if err != nil {
		// Run a dummy bcrypt comparison so this path takes roughly the
		// same wall-clock time as a real password check. Without this,
		// an attacker can enumerate valid emails by measuring response
		// latency.
		_ = helpers.ValidateHash(inDom.Password, dummyBcryptHash)
		return entities.UserDomain{}, apperror.Unauthorized("invalid email or password")
	}

	if !userDomain.Active {
		return entities.UserDomain{}, apperror.Forbidden("account is not activated")
	}

	if !helpers.ValidateHash(inDom.Password, userDomain.Password) {
		return entities.UserDomain{}, apperror.Unauthorized("invalid email or password")
	}

	isAdmin := userDomain.RoleID == constants.AdminID
	pair, err := userUC.jwtService.GenerateTokenPair(userDomain.ID, isAdmin, userDomain.Email)
	if err != nil {
		return entities.UserDomain{}, apperror.InternalCause(fmt.Errorf("generate token: %w", err))
	}
	if err := userUC.rememberRefresh(ctx, pair); err != nil {
		// If Redis is unavailable we'd rather fail login than issue a
		// refresh token the /refresh endpoint can't verify.
		return entities.UserDomain{}, apperror.InternalCause(fmt.Errorf("persist refresh: %w", err))
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
func (userUC *userUsecase) Refresh(ctx context.Context, refreshToken string) (entities.UserDomain, error) {
	claims, err := userUC.jwtService.ParseRefreshToken(refreshToken)
	if err != nil {
		return entities.UserDomain{}, apperror.Unauthorized("invalid refresh token")
	}

	// Verify the jti is still live server-side; logout / previous
	// rotation would have removed it.
	if _, err := userUC.redisCache.Get(ctx, refreshKey(claims.ID)); err != nil {
		return entities.UserDomain{}, apperror.Unauthorized("refresh token has been revoked")
	}

	// Fresh identity lookup so revoked / deactivated accounts stop
	// getting new access tokens even while their refresh is live.
	userDomain, err := userUC.repo.GetByEmail(ctx, &entities.UserDomain{Email: claims.Email})
	if err != nil {
		return entities.UserDomain{}, apperror.Unauthorized("user no longer exists")
	}
	if !userDomain.Active {
		return entities.UserDomain{}, apperror.Forbidden("account is not activated")
	}

	isAdmin := userDomain.RoleID == constants.AdminID
	pair, err := userUC.jwtService.GenerateTokenPair(userDomain.ID, isAdmin, userDomain.Email)
	if err != nil {
		return entities.UserDomain{}, apperror.InternalCause(fmt.Errorf("generate token: %w", err))
	}

	// Rotate: remove the old jti, record the new one. Do this after
	// the new pair is minted so a mint failure doesn't leave the user
	// with no valid refresh token at all.
	if err := userUC.rememberRefresh(ctx, pair); err != nil {
		return entities.UserDomain{}, apperror.InternalCause(fmt.Errorf("persist refresh: %w", err))
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
	email = normalizeEmail(email)
	domain, err := userUC.repo.GetByEmail(ctx, &entities.UserDomain{Email: email})
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
	email = normalizeEmail(email)
	domain, err := userUC.repo.GetByEmail(ctx, &entities.UserDomain{Email: email})
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
	if err = userUC.repo.ChangeActiveUser(ctx, &entities.UserDomain{ID: domain.ID, Active: true}); err != nil {
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

func (userUC *userUsecase) GetByEmail(ctx context.Context, email string) (entities.UserDomain, error) {
	email = normalizeEmail(email)
	// check in-memory cache first
	cacheKey := fmt.Sprintf("user/%s", email)
	if val := userUC.ristrettoCache.Get(cacheKey); val != nil {
		if cached, ok := val.(entities.UserDomain); ok {
			observability.ObserveCacheOp("ristretto", "get", "hit")
			return cached, nil
		}
		observability.ObserveCacheOp("ristretto", "get", "error")
		logger.Info("cache type assertion failed, fetching from DB", logrus.Fields{constants.LoggerCategory: constants.LoggerCategoryCache})
	} else {
		observability.ObserveCacheOp("ristretto", "get", "miss")
	}

	// Coalesce concurrent misses for the same email — without this,
	// N goroutines all racing on a cold cache fan out into N DB
	// round-trips. singleflight runs the fn() once per key and hands
	// the result to every joiner.
	v, err, _ := userUC.userByEmailGroup.Do(email, func() (any, error) {
		user, repoErr := userUC.repo.GetByEmail(ctx, &entities.UserDomain{Email: email})
		if repoErr != nil {
			return entities.UserDomain{}, repoErr
		}
		userUC.ristrettoCache.Set(cacheKey, user)
		observability.ObserveCacheOp("ristretto", "set", "ok")
		return user, nil
	})
	if err != nil {
		return entities.UserDomain{}, apperror.NotFound("email not found")
	}
	return v.(entities.UserDomain), nil
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
