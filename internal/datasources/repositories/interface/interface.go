// Package _interface holds the gateway abstractions for every domain
// in the repositories layer. The package name is "_interface" because
// "interface" is a Go reserved keyword and can't be used as an
// identifier directly; the leading underscore keeps it a valid
// identifier without changing the conceptual meaning.
//
// Concrete adapters (postgres/, future mongo/, etc.) implement these
// interfaces and live as siblings of this package. The use case
// layer depends only on this package — never on a concrete adapter
// — so swapping the storage engine doesn't ripple into business
// code.
package _interface

import (
	"context"

	"github.com/go-rest-boilerplate/internal/business/domain"
)

type PlayerRepository interface {
	Store(ctx context.Context, in *domain.Player) (domain.Player, error)
}
