// Package events is an in-process event bus. A module publishes an event, and the handlers other modules
// subscribed to its type run in the publisher's goroutine.
//
// Delivery is at most once: an event is lost if the process stops between the publisher's change and its
// handlers, or when a handler fails, since nothing retries it. A handler therefore never assumes it saw every
// event, and an event that must not be lost goes through an outbox instead.
package events

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"runtime/debug"
	"slices"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Bus delivers events to the handlers subscribed to their type. It is safe for concurrent use. Subscribe every
// handler at start-up, before the first Publish: a handler added later misses the events published before.
type Bus struct {
	log    *slog.Logger
	tracer trace.Tracer

	mu       sync.RWMutex
	handlers map[reflect.Type][]handler
}

type handler struct {
	name string
	run  func(ctx context.Context, event any) error
}

// NewBus returns a Bus that logs the failures of its handlers to log and traces each handler with tracerProvider.
func NewBus(log *slog.Logger, tracerProvider trace.TracerProvider) *Bus {
	return &Bus{
		log:      log,
		tracer:   tracerProvider.Tracer("github.com/snykk/go-rest-boilerplate/internal/platform/events"),
		handlers: make(map[reflect.Type][]handler),
	}
}

// Subscribe adds handle, named name in log records and spans, to the handlers of the events whose dynamic type is
// exactly E; a pointer to E is another type. E must be a concrete type: a handler of an interface type never
// runs. The handlers of one type run in the order they subscribed. handle runs on every publishing goroutine at
// once, so it must be safe for concurrent use.
func Subscribe[E any](bus *Bus, name string, handle func(ctx context.Context, event E) error) {
	eventType := reflect.TypeFor[E]()
	run := func(ctx context.Context, event any) error { return handle(ctx, event.(E)) }

	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.handlers[eventType] = append(slices.Clip(bus.handlers[eventType]), handler{name: name, run: run})
}

// Publish runs every handler of event's dynamic type in the caller's goroutine, each in a span named after the
// handler, child of the span in ctx, with the attribute event.type. Each handler gets ctx, so ctx's deadline
// and cancellation apply to it. A handler's error or panic is logged at ERROR with the handler's name and the
// event's type, and the handlers after it still run; nothing retries the event. Publish never fails.
func (b *Bus) Publish(ctx context.Context, event any) {
	b.mu.RLock()
	handlers := b.handlers[reflect.TypeOf(event)]
	b.mu.RUnlock()

	eventType := fmt.Sprintf("%T", event)
	for _, subscriber := range handlers {
		b.deliver(ctx, subscriber, eventType, event)
	}
}

func (b *Bus) deliver(ctx context.Context, subscriber handler, eventType string, event any) {
	ctx, span := b.tracer.Start(ctx, subscriber.name, trace.WithAttributes(attribute.String("event.type", eventType)))
	defer span.End()
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		span.SetStatus(codes.Error, "handler panicked")
		b.log.LogAttrs(ctx, slog.LevelError, "event handler panicked", slog.String("handler", subscriber.name),
			slog.String("event.type", eventType), slog.Any("panic", recovered), slog.String("stack", string(debug.Stack())))
	}()

	if err := subscriber.run(ctx, event); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "handler failed")
		b.log.LogAttrs(ctx, slog.LevelError, "event handler failed", slog.String("handler", subscriber.name),
			slog.String("event.type", eventType), slog.String("error", err.Error()))
	}
}
