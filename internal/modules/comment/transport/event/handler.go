package event

import "github.com/snykk/go-rest-boilerplate/internal/platform/events"

// handler reacts to the events the comment module subscribes to.
type handler struct {
	useCases UseCases
}

// Subscribe makes bus run the comment use cases when another module announces an event: DeleteForArticle when
// the article module announces ArticleDeleted. Call it at start-up, before the first Publish. Delivery is at
// most once, so a comment is left behind when its article's event is lost or its use case fails.
func Subscribe(bus *events.Bus, useCases UseCases) {
	h := handler{useCases: useCases}
	events.Subscribe(bus, "comment.article-deleted", h.articleDeleted)
}
