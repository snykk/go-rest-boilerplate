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
}
