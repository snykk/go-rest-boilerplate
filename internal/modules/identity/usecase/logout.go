package usecase

import "context"

// Logout ends the sign-in the refresh token belongs to: it revokes the token's session family, the token itself
// and every token rotated from the same sign-in, so a refresh running at the same time cannot leave a session
// alive. It returns nil for a token no session has, so a repeated call is harmless, and the store's error when
// it fails.
func (u *UserUsecase) Logout(ctx context.Context, refreshToken string) error {
	return u.sessions.Revoke(ctx, hashOf(refreshToken), u.clock.Now())
}
