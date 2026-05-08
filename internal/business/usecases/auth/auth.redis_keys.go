package auth

import "fmt"

// Redis key prefixes used by the auth domain. Centralized so a typo
// can't silently desync a writer from its reader, and so adapters
// outside this package (notably the auth middleware) reuse the exact
// same names instead of re-implementing the format string.
const (
	prefixRefresh        = "refresh:"
	prefixUserOTP        = "user_otp:"
	prefixOTPAttempts    = "otp_attempts:"
	prefixLoginAttempts  = "login_attempts:"
	prefixForgotAttempts = "forgot_attempts:"
	prefixPasswordReset  = "pwd_reset:"
	prefixUserResetIndex = "pwd_reset_user:"
	prefixPasswordCutoff = "pwd_cutoff:"
)

// RefreshKey scopes refresh-token jti entries; absence ⇒ revoked.
func RefreshKey(jti string) string {
	return fmt.Sprintf("%s%s", prefixRefresh, jti)
}

// UserOTPKey holds the live 6-digit OTP for an inactive account.
func UserOTPKey(email string) string {
	return fmt.Sprintf("%s%s", prefixUserOTP, email)
}

// OTPAttemptsKey counts failed VerifyOTP attempts per email.
func OTPAttemptsKey(email string) string {
	return fmt.Sprintf("%s%s", prefixOTPAttempts, email)
}

// LoginAttemptsKey counts failed Login attempts per email for the
// brute-force lockout window.
func LoginAttemptsKey(email string) string {
	return fmt.Sprintf("%s%s", prefixLoginAttempts, email)
}

// ForgotAttemptsKey rate-limits /password/forgot per email.
func ForgotAttemptsKey(email string) string {
	return fmt.Sprintf("%s%s", prefixForgotAttempts, email)
}

// PasswordResetKey holds a one-shot reset token's user ID; the token
// itself is the suffix.
func PasswordResetKey(token string) string {
	return fmt.Sprintf("%s%s", prefixPasswordReset, token)
}

// UserResetIndexKey is the per-user reverse index from user ID to the
// token currently outstanding, used so issuing a fresh reset link
// invalidates any prior live token.
func UserResetIndexKey(userID string) string {
	return fmt.Sprintf("%s%s", prefixUserResetIndex, userID)
}

// TokenCutoffKey holds the unix-seconds cutoff after which any access
// token issued for this user is considered revoked. Auth middleware
// reads this on every authenticated request; ChangePassword and
// ResetPassword write it.
func TokenCutoffKey(userID string) string {
	return fmt.Sprintf("%s%s", prefixPasswordCutoff, userID)
}
