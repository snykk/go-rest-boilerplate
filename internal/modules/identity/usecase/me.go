package usecase

import "context"

// Me returns the user with the given ID. It returns domain.ErrNotFound for an unknown ID and the store's
// error when it fails.
func (u *UserUsecase) Me(ctx context.Context, userID string) (UserView, error) {
	user, err := u.users.ByID(ctx, userID)
	if err != nil {
		return UserView{}, err
	}
	return viewOf(user), nil
}
