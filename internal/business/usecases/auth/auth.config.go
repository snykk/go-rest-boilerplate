package auth

import "time"

// Config is the slice of configuration the auth use case needs.
// Injecting it via NewUsecase keeps this package free of any
// dependency on internal/config — the composition root translates
// the env config into the shape the auth domain cares about.
type Config struct {
	// OTPMaxAttempts is the lockout threshold for VerifyOTP. After
	// this many failures within the OTP window the email is locked
	// out even with the correct code.
	OTPMaxAttempts int
	// OTPTTL is how long the OTP code (and its attempt counter)
	// stay live in Redis before expiring.
	OTPTTL time.Duration
	// PasswordResetTTL is how long a forgot-password token stays
	// usable. 30 minutes is a reasonable default — long enough for an
	// email round-trip, short enough that a leaked link expires fast.
	PasswordResetTTL time.Duration
	// BcryptCost is forwarded to domain.User.ChangePassword on
	// password change/reset. Caller (DI) injects from app config.
	BcryptCost int
	// LoginMaxAttempts is the lockout threshold for /auth/login. After
	// this many failures (per email, within LoginLockoutTTL) the email
	// is locked out for the remaining window even on a correct
	// password — defeats slow brute-force from distributed IPs that
	// per-IP rate limiting can't see.
	LoginMaxAttempts int
	// LoginLockoutTTL is how long the lockout window lasts and how
	// long the per-email failure counter stays live. 15m is a
	// reasonable default; long enough to defeat brute force, short
	// enough that a legitimate user with a typo isn't permanently
	// blocked.
	LoginLockoutTTL time.Duration
	// ForgotMaxAttempts caps how many /password/forgot calls one
	// email can trigger inside ForgotLockoutTTL. Defends against
	// abuse of the mailer queue (DOS via outbound email spam) and
	// against attacker-driven reset-token rotation.
	ForgotMaxAttempts int
	// ForgotLockoutTTL is the rate-limit window for /password/forgot.
	ForgotLockoutTTL time.Duration
}
