package identity

import (
	"context"
	"errors"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/contract"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

//go:generate go tool mockgen -source=api.go -destination=api_mock_test.go -package=identity_test

// UserReader is the use case API needs: looking a user up by ID.
type UserReader interface {
	// Me returns the user with the given ID, or domain.ErrNotFound.
	Me(ctx context.Context, userID string) (usecase.UserView, error)
}

// API is what other modules may call on the identity module, each through a gateway of its own. It is safe for
// concurrent use.
type API struct {
	users UserReader
}

// NewAPI returns an API over users.
func NewAPI(users UserReader) *API {
	return &API{users: users}
}

// User returns the user with the given ID. It returns contract.ErrNotFound for an unknown ID, including one
// that is not a UUID, and the store's error when it fails.
func (a *API) User(ctx context.Context, id string) (contract.User, error) {
	view, err := a.users.Me(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return contract.User{}, contract.ErrNotFound
	}
	if err != nil {
		return contract.User{}, err
	}
	return contract.User{ID: view.ID, DisplayName: view.DisplayName}, nil
}
