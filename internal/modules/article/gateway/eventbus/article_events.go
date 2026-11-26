// Package eventbus announces article events on the application's in-process event bus.
package eventbus

import (
	"context"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/contract"
	"github.com/snykk/go-rest-boilerplate/internal/platform/events"
)

// ArticleEvents implements usecase.ArticleEvents over an events.Bus. It is safe for concurrent use.
type ArticleEvents struct {
	bus *events.Bus
}

// NewArticleEvents returns ArticleEvents publishing on bus.
func NewArticleEvents(bus *events.Bus) *ArticleEvents {
	return &ArticleEvents{bus: bus}
}

// Deleted publishes event, as a value, to the handlers of contract.ArticleDeleted, which run before it returns.
func (a *ArticleEvents) Deleted(ctx context.Context, event contract.ArticleDeleted) {
	a.bus.Publish(ctx, event)
}
