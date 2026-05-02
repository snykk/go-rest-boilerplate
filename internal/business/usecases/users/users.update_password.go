package users

import (
	"context"
	"fmt"

	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
)

// UpdatePassword persists the already-hashed password + revocation
// timestamp on the supplied user. Caller is expected to invoke
// domain.User.ChangePassword first to populate Password +
// PasswordChangedAt + UpdatedAt before calling this.
func (uc *usecase) UpdatePassword(ctx context.Context, user *domain.User) error {
	if user == nil || user.ID == "" {
		return apperror.BadRequest("user id required")
	}
	if err := uc.repo.UpdatePassword(ctx, user); err != nil {
		return mapRepoError(err, fmt.Sprintf("update password for %s", user.ID))
	}
	return nil
}
