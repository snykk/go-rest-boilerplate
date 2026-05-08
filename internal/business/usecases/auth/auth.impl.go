package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/users"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/caches"
	"github.com/snykk/go-rest-boilerplate/pkg/jwt"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
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

// tokenCutoffTTL bounds how long the cutoff signal needs to live.
// Access tokens expire after at most uc.cfg.JWTExpired hours, so 24h
// comfortably outlives any in-flight access token. Refresh-token
// revocation lives in the DB (User.TokensRevokedBefore) and isn't
// affected by this TTL.
const tokenCutoffTTL = 24 * time.Hour

// recordTokenCutoff publishes a "tokens issued before this instant
// are revoked" marker that AuthMiddleware checks on every request,
// so a leaked access token stops working as soon as the user rotates
// their password instead of lingering until natural expiry.
func (uc *usecase) recordTokenCutoff(ctx context.Context, userID string, when time.Time) {
	key := TokenCutoffKey(userID)
	if err := uc.redisCache.Set(ctx, key, fmt.Sprintf("%d", when.Unix())); err != nil {
		logger.ErrorWithContext(ctx, "auth: failed to write token cutoff (non-fatal)", logger.Fields{
			"step":    "redis_set_token_cutoff",
			"error":   err.Error(),
			"user_id": userID,
		})
		return
	}
	_ = uc.redisCache.Expire(ctx, key, tokenCutoffTTL)
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
	if err := uc.redisCache.Set(ctx, RefreshKey(pair.RefreshJTI), pair.RefreshJTI); err != nil {
		return err
	}
	// Set() applies the cache-wide expires in minutes; override
	// explicitly so each refresh token has its own TTL.
	return uc.redisCache.Expire(ctx, RefreshKey(pair.RefreshJTI), ttl)
}
