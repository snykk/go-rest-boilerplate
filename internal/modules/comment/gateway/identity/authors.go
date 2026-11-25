// Package identity tells the comment module who the authors of comments are, by asking the identity module's
// API: an anti-corruption layer that keeps that module's model out of this one.
package identity

import (
	"context"
	"errors"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/contract"
)

//go:generate go tool mockgen -source=authors.go -destination=authors_mock_test.go -package=identity_test

// API is the part of the identity module's API that Authors asks.
type API interface {
	// User returns the user with the given ID, or contract.ErrNotFound.
	User(ctx context.Context, id string) (contract.User, error)
}

// Authors implements usecase.Authors over the identity module's API. It is safe for concurrent use when its API
// is.
type Authors struct {
	api API
}

var _ usecase.Authors = (*Authors)(nil)

// NewAuthors returns Authors that ask api.
func NewAuthors(api API) *Authors {
	return &Authors{api: api}
}

// DisplayName returns the display name of the user with the given ID. It returns domain.ErrAuthorNotFound for a
// user the API reports as contract.ErrNotFound, and the API's other errors as they are.
func (a *Authors) DisplayName(ctx context.Context, userID string) (string, error) {
	user, err := a.api.User(ctx, userID)
	if errors.Is(err, contract.ErrNotFound) {
		return "", domain.ErrAuthorNotFound
	}
	if err != nil {
		return "", err
	}
	return user.DisplayName, nil
}
