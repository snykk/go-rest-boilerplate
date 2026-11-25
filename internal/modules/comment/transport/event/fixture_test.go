package event_test

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/transport/event"
	"github.com/snykk/go-rest-boilerplate/internal/platform/events"
)

const articleID = "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c45"

// errStoreDown is a store failing on its own.
var errStoreDown = errors.New("delete comments: connection refused")

// fixture is an event bus whose failures are logged into logs, with the handlers of the comment module
// subscribed over a mocked UseCases.
type fixture struct {
	useCases *MockUseCases
	logs     *bytes.Buffer
	bus      *events.Bus
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{useCases: NewMockUseCases(gomock.NewController(t)), logs: &bytes.Buffer{}}
	f.bus = events.NewBus(slog.New(slog.NewJSONHandler(f.logs, nil)), noop.NewTracerProvider())
	event.Subscribe(f.bus, f.useCases)
	return f
}
