package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/mailer"
)

// usecase carries the dependencies and any cross-method state. Each
// method lives in its own file so PR diffs stay surgical when a
// single behavior changes.
type usecase struct {
	users      users.Usecase
	jwtService jwt.JWTService
	mailer     mailer.OTPMailer
	redisCache caches.RedisCache
	cfg        Config
}

// NewUsecase wires the auth flows. It depends on users.Usecase (the
// User bounded-context input port) for identity reads / writes, and
// on infrastructure (jwt, redis, mailer) for the auth-specific bits.
func NewUsecase(usersUC users.Usecase, jwtService jwt.JWTService, otpMailer mailer.OTPMailer, redisCache caches.RedisCache, cfg Config) Usecase {
	return &usecase{
		users:      usersUC,
		jwtService: jwtService,
		mailer:     otpMailer,
		redisCache: redisCache,
		cfg:        cfg,
	}
}

// dummyBcryptHash is a pre-computed bcrypt hash used to mask the
// timing difference between "user not found" and "wrong password"
// branches in Login. Comparing an arbitrary password against it
// takes the same ~100ms a real bcrypt comparison does, preventing
// user enumeration via response latency.
const dummyBcryptHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// normalizeEmail mirrors the same helper in the users package; auth
// has its own copy so it doesn't have to expose normalization across
// the bounded-context boundary.
func normalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// refreshKey scopes refresh-token jti entries so they don't collide
// with OTP keys in Redis.
func refreshKey(jti string) string { return fmt.Sprintf("refresh:%s", jti) }

// otpAttemptsKey returns the Redis key that tracks failed VerifyOTP
// attempts for an email, scoped separately from the OTP code itself.
func otpAttemptsKey(email string) string {
	return fmt.Sprintf("otp_attempts:%s", email)
}

// rememberRefresh stores the refresh jti in Redis with a TTL matching
// the refresh token's exp. /refresh and /logout treat absence here as
// "revoked", which is how logout works without an access-token
// blacklist.
func (uc *usecase) rememberRefresh(ctx context.Context, pair jwt.TokenPair) error {
	ttl := time.Until(pair.RefreshExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("refresh token already expired")
	}
	if err := uc.redisCache.Set(ctx, refreshKey(pair.RefreshJTI), pair.RefreshJTI); err != nil {
		return err
	}
	// Set() applies the cache-wide expires in minutes; override
	// explicitly so each refresh token has its own TTL.
	return uc.redisCache.Expire(ctx, refreshKey(pair.RefreshJTI), ttl)
}
