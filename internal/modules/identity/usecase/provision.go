package usecase

import "context"

// ProvisionInput is the account Provision creates.
type ProvisionInput struct {
	Email, Password, DisplayName string
	Roles                        []string
}

// Provision stores a new user with the given roles whose email is already verified. It is for the seed data of
// local development, which creates the editor this way; in production no code grants a role, and an operator
// grants the editor role to a registered user with SQL. It returns domain.ErrInvalidEmail and
// domain.ErrWeakPassword before the password is hashed, domain.ErrInvalidDisplayName before the store is asked,
// domain.ErrEmailTaken when another user has the email, and the store's or the hasher's error when they fail.
func (u *UserUsecase) Provision(ctx context.Context, input ProvisionInput) (UserView, error) {
	now := u.clock.Now()
	user, err := u.newUser(input.Email, input.Password, input.DisplayName, input.Roles, now)
	if err != nil {
		return UserView{}, err
	}

	user.Verify(now)
	if err := u.users.Create(ctx, user); err != nil {
		return UserView{}, err
	}
	return viewOf(user), nil
}
