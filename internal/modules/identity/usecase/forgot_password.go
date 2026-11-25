package usecase

import "context"

// ForgotPassword stores a new password reset token for the verified user with the given email, replacing the
// one they held, and mails it. It returns nil, and mails nothing, for an email that is invalid, that no user
// has or whose user is not verified, so the answer does not tell whether an account exists. While the token the
// user holds is younger than Settings.ResendCooldown, it keeps that token, stores and mails nothing, and returns
// nil too, so the answer does not tell that either. It returns the store's error when it fails.
func (u *UserUsecase) ForgotPassword(ctx context.Context, email string) error {
	user, found, err := u.lookup(ctx, email)
	if err != nil {
		return err
	}
	if !found || !user.Verified() {
		return nil
	}

	token := u.random.Token()
	secret := newSecret(user.ID(), token, u.clock.Now(), u.settings.ResetTTL)
	stored, err := u.resets.Put(ctx, secret, u.settings.ResendCooldown)
	if err != nil || !stored {
		return err
	}

	u.mail.PasswordReset(ctx, user.Email(), token)
	return nil
}
