package identity

import (
	"fmt"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/platform/auth/jwt"
	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
)

const (
	// minSecretBytes is the shortest JWT secret accepted: HS256 needs a key of at least its 256 bits.
	minSecretBytes = 32
	// minBcryptCost and maxBcryptCost bound the bcrypt cost: below 10 a hash is cheap to guess at, above 14 a
	// sign-in takes seconds.
	minBcryptCost, maxBcryptCost = 10, 14
	// minLifetime is the shortest lifetime of a token or a code, 1m; the messages give lifetimes in minutes.
	minLifetime = time.Minute
	// jwtLeeway is how far past its expiry an access token is still accepted, for clock drift between the
	// replicas that sign and verify it.
	jwtLeeway = 10 * time.Second
)

// Config holds the identity module's own settings, read from IDENTITY_* variables.
type Config struct {
	// JWTSecret signs and verifies access tokens: at least 32 bytes, kept secret.
	JWTSecret string
	// JWTIssuer and JWTAudience are the iss and aud claims of an access token, which the verifier checks.
	JWTIssuer, JWTAudience string
	// AccessTTL is the lifetime of an access token, which stays valid until then after a sign-out.
	AccessTTL time.Duration
	// RefreshTTL is the lifetime of a refresh token.
	RefreshTTL time.Duration
	// RefreshReuseGrace is how long after a refresh token was spent presenting it again is taken as a client
	// that missed the answer, not as a stolen token.
	RefreshReuseGrace time.Duration
	// CodeTTL is the lifetime of an e-mail verification code.
	CodeTTL time.Duration
	// CodeAttempts is how many wrong guesses end a verification code.
	CodeAttempts int
	// ResetTTL is the lifetime of a password reset token.
	ResetTTL time.Duration
	// ResendCooldown is how long after a verification code or a reset token was issued to a user a request for
	// another one is answered as usual but issues and mails nothing.
	ResendCooldown time.Duration
	// BcryptCost is the cost passwords are hashed at, from 10 to 14.
	BcryptCost int
	// LoginAttempts is how many failed sign-ins in a row lock an account.
	LoginAttempts int
	// Lockout is how long a locked account stays locked.
	Lockout time.Duration
}

// LoadConfig reads from source IDENTITY_JWT_SECRET (required, at least 32 bytes), IDENTITY_JWT_ISSUER and
// IDENTITY_JWT_AUDIENCE (default go-rest-boilerplate), IDENTITY_ACCESS_TTL (15m), IDENTITY_REFRESH_TTL (720h),
// IDENTITY_REFRESH_REUSE_GRACE (10s), IDENTITY_CODE_TTL (10m), IDENTITY_CODE_ATTEMPTS (5), IDENTITY_RESET_TTL
// (30m), IDENTITY_RESEND_COOLDOWN (60s), IDENTITY_BCRYPT_COST (12, from 10 to 14), IDENTITY_LOGIN_ATTEMPTS (5)
// and IDENTITY_LOCKOUT (15m). The lifetimes of access tokens, refresh tokens, codes and reset tokens must be at
// least a minute, the other durations and the counts positive. It reports every problem in one error.
func LoadConfig(source config.Source) (Config, error) {
	reader := config.NewReader(source)
	settings := Config{
		JWTSecret:         reader.Required("IDENTITY_JWT_SECRET"),
		JWTIssuer:         reader.String("IDENTITY_JWT_ISSUER", "go-rest-boilerplate"),
		JWTAudience:       reader.String("IDENTITY_JWT_AUDIENCE", "go-rest-boilerplate"),
		AccessTTL:         reader.Duration("IDENTITY_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:        reader.Duration("IDENTITY_REFRESH_TTL", 720*time.Hour),
		RefreshReuseGrace: reader.Duration("IDENTITY_REFRESH_REUSE_GRACE", 10*time.Second),
		CodeTTL:           reader.Duration("IDENTITY_CODE_TTL", 10*time.Minute),
		CodeAttempts:      int(reader.PositiveInt("IDENTITY_CODE_ATTEMPTS", 5)),
		ResetTTL:          reader.Duration("IDENTITY_RESET_TTL", 30*time.Minute),
		ResendCooldown:    reader.Duration("IDENTITY_RESEND_COOLDOWN", time.Minute),
		BcryptCost:        int(reader.PositiveInt("IDENTITY_BCRYPT_COST", 12)),
		LoginAttempts:     int(reader.PositiveInt("IDENTITY_LOGIN_ATTEMPTS", 5)),
		Lockout:           reader.Duration("IDENTITY_LOCKOUT", 15*time.Minute),
	}
	if settings.JWTSecret != "" && len(settings.JWTSecret) < minSecretBytes {
		reader.Fail("IDENTITY_JWT_SECRET", fmt.Sprintf("must be at least %d bytes", minSecretBytes))
	}
	if settings.BcryptCost < minBcryptCost || settings.BcryptCost > maxBcryptCost {
		reader.Fail("IDENTITY_BCRYPT_COST", fmt.Sprintf("must be from %d to %d", minBcryptCost, maxBcryptCost))
	}
	lifetimes := []struct {
		key   string
		value time.Duration
	}{
		{"IDENTITY_ACCESS_TTL", settings.AccessTTL},
		{"IDENTITY_REFRESH_TTL", settings.RefreshTTL},
		{"IDENTITY_CODE_TTL", settings.CodeTTL},
		{"IDENTITY_RESET_TTL", settings.ResetTTL},
	}
	for _, lifetime := range lifetimes {
		if lifetime.value < minLifetime {
			reader.Fail(lifetime.key, "must be at least 1m")
		}
	}

	if err := reader.Err(); err != nil {
		return Config{}, err
	}
	return settings, nil
}

// JWT returns the settings the access token's issuer and verifier share: the secret, the issuer and the
// audience, the access token's lifetime, and a leeway of 10 seconds for clock drift between replicas.
func (c Config) JWT() jwt.Config {
	return jwt.Config{
		Secret:   []byte(c.JWTSecret),
		Issuer:   c.JWTIssuer,
		Audience: c.JWTAudience,
		TTL:      c.AccessTTL,
		Leeway:   jwtLeeway,
	}
}
