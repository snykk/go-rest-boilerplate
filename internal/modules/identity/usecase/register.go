package usecase

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/contract"
)

// RegisterInput is the account Register creates.
type RegisterInput struct{ Email, Password, DisplayName string }

// Register stores a new unverified user with the role contract.RoleReader and its verification code in one
// unit of work, then mails the code. It returns domain.ErrInvalidEmail and domain.ErrWeakPassword before the
// password is hashed, domain.ErrInvalidDisplayName before the store is asked, domain.ErrEmailTaken when another
// user has the email, in which case it mails nothing, and the store's or the hasher's error when they fail.
func (u *UserUsecase) Register(ctx context.Context, input RegisterInput) (UserView, error) {
	now := u.clock.Now()
	user, err := u.newUser(input.Email, input.Password, input.DisplayName, []string{contract.RoleReader}, now)
	if err != nil {
		return UserView{}, err
	}

	code := u.random.Code()
	secret := newSecret(user.ID(), code, now, u.settings.CodeTTL)
	err = u.unitOfWork.Do(ctx, func(repositories Repositories) error {
		if err := repositories.Users.Create(ctx, user); err != nil {
			return err
		}
		_, err := repositories.Verifications.Put(ctx, secret, u.settings.ResendCooldown)
		return err
	})
	if err != nil {
		return UserView{}, err
	}

	u.mail.Verification(ctx, user.Email(), code)
	return viewOf(user), nil
}
