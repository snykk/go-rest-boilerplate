package usecase

import "context"

// ResendVerification stores a new verification code for the unverified user with the given email, replacing
// the code they held, and mails it. It returns nil, and mails nothing, for an email that is invalid, that no
// user has or whose user is verified, so the answer does not tell whether an account exists. While the code the
// user holds is younger than Settings.ResendCooldown, it keeps that code, stores and mails nothing, and returns
// nil too, so the answer does not tell that either. It returns the store's error when it fails.
func (u *UserUsecase) ResendVerification(ctx context.Context, email string) error {
	user, found, err := u.lookup(ctx, email)
	if err != nil {
		return err
	}
	if !found || user.Verified() {
		return nil
	}

	code := u.random.Code()
	secret := newSecret(user.ID(), code, u.clock.Now(), u.settings.CodeTTL)
	stored, err := u.verifications.Put(ctx, secret, u.settings.ResendCooldown)
	if err != nil || !stored {
		return err
	}

	u.mail.Verification(ctx, user.Email(), code)
	return nil
}
