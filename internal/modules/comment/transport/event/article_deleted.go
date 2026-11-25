package event

import (
	"context"
	"fmt"

	articlecontract "github.com/snykk/go-rest-boilerplate/internal/modules/article/contract"
)

// articleDeleted removes the comments of the deleted article. It returns the use case's error, wrapped with the
// article's ID, for the bus to log.
func (h handler) articleDeleted(ctx context.Context, event articlecontract.ArticleDeleted) error {
	if err := h.useCases.DeleteForArticle(ctx, event.ID); err != nil {
		return fmt.Errorf("delete the comments of article %s: %w", event.ID, err)
	}
	return nil
}
