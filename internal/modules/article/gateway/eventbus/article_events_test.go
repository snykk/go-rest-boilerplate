package eventbus_test

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/contract"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/gateway/eventbus"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/events"
)

var _ usecase.ArticleEvents = (*eventbus.ArticleEvents)(nil)

// Deleted publishes the event, as a value, to the bus's handlers.
func TestArticleEventsDeleted(t *testing.T) {
	tCases := []struct {
		name  string
		event contract.ArticleDeleted
	}{
		{
			name: "the event reaches a handler of contract.ArticleDeleted once",
			event: contract.ArticleDeleted{
				ID: "a1",
				At: time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			bus := events.NewBus(slog.New(slog.NewTextHandler(io.Discard, nil)), noop.NewTracerProvider())
			var got []contract.ArticleDeleted
			events.Subscribe(bus, "record", func(_ context.Context, event contract.ArticleDeleted) error {
				got = append(got, event)
				return nil
			})

			eventbus.NewArticleEvents(bus).Deleted(t.Context(), tc.event)

			if want := []contract.ArticleDeleted{tc.event}; !reflect.DeepEqual(got, want) {
				t.Errorf("handled = %v, want %v", got, want)
			}
		})
	}
}
