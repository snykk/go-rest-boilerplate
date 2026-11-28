package mailer

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Queue is an Enqueuer that holds messages in memory and sends them one at a time, in order, through a Sender
// while Run is running. Once Run's context ends, the queue drains: it goes on taking and sending messages for
// up to its drain timeout. Once the drain ends, Enqueue drops every message. Messages are not persisted: those
// still queued when the process dies are lost, so a message that must not be lost needs a durable path such as
// an outbox instead. Every message Enqueue is given is sent or logged: one Enqueue drops, or whose send fails,
// is named by its Kind in a log record, and those still held when the drain timeout ends are dropped and
// counted in one. A Queue is safe for concurrent use.
type Queue struct {
	sender       Sender
	log          *slog.Logger
	messages     chan Message
	drainTimeout time.Duration

	mu      sync.RWMutex
	stopped bool
}

// NewQueue returns a Queue that sends through sender and logs to log, and holds up to size messages; a size
// below 1 counts as 1, and one above MaxQueueSize as MaxQueueSize. Once Run's context ends, the queue drains:
// it goes on sending for up to drainTimeout.
func NewQueue(sender Sender, log *slog.Logger, size int, drainTimeout time.Duration) *Queue {
	return &Queue{
		sender:       sender,
		log:          log,
		messages:     make(chan Message, min(max(size, 1), MaxQueueSize)),
		drainTimeout: drainTimeout,
	}
}

// Enqueue adds message to the queue and returns at once. When the queue is full, or once its drain has ended
// and Run is returning or has returned, it drops the message and logs a warning with the message's Kind.
func (q *Queue) Enqueue(ctx context.Context, message Message) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.stopped {
		q.log.LogAttrs(ctx, slog.LevelWarn, "mail queue stopped, message dropped", slog.String("kind", message.Kind))
		return
	}

	select {
	case q.messages <- message:
	default:
		q.log.LogAttrs(ctx, slog.LevelWarn, "mail queue full", slog.String("kind", message.Kind))
	}
}

// Run sends queued messages until ctx ends, then drains the queue and returns nil. The end of ctx cancels no
// send: while the queue drains, the send in flight and the ones after it go on, each bounded by the Sender's
// own timeout, until the drain timeout has passed since ctx ended. Then the send in flight is cancelled. The
// drain ends once the queue is empty or the drain timeout has passed: Enqueue drops every message from then on,
// and Run sends what came in before, within the same timeout. The messages it still holds after that are
// dropped and their number logged, so Run returns soon after the drain timeout at the latest. A send that fails
// is logged with the message's Kind and the error, and the next message goes on. Run is called once.
func (q *Queue) Run(ctx context.Context) error {
	sendCtx, stopSending := context.WithCancel(context.WithoutCancel(ctx))
	defer stopSending()
	context.AfterFunc(ctx, func() { q.stopAfterDrain(sendCtx, stopSending) })

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
		case message := <-q.messages:
			q.send(sendCtx, message)
		}
	}

	q.sendHeld(sendCtx)
	q.stop()
	q.sendHeld(sendCtx)
	q.dropHeld(sendCtx)
	return nil
}

// stopAfterDrain calls stopSending once the drain timeout has passed, or once sending has ended, whichever
// comes first.
func (q *Queue) stopAfterDrain(sending context.Context, stopSending context.CancelFunc) {
	deadline, cancel := context.WithTimeout(sending, q.drainTimeout)
	defer cancel()
	<-deadline.Done()
	stopSending()
}

// stop makes Enqueue drop every message from now on. A message Enqueue took before is in the queue by the time
// stop returns.
func (q *Queue) stop() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.stopped = true
}

// sendHeld sends the messages the queue holds until it is empty or ctx ends.
func (q *Queue) sendHeld(ctx context.Context) {
	for ctx.Err() == nil {
		select {
		case message := <-q.messages:
			q.send(ctx, message)
		default:
			return
		}
	}
}

// dropHeld drops the messages a stopped queue still holds and logs how many, when there are any.
func (q *Queue) dropHeld(ctx context.Context) {
	unsent := len(q.messages)
	if unsent == 0 {
		return
	}
	for range unsent {
		<-q.messages
	}
	q.log.LogAttrs(ctx, slog.LevelWarn, "mail queue stopped with messages unsent", slog.Int("unsent", unsent))
}

func (q *Queue) send(ctx context.Context, message Message) {
	if err := q.sender.Send(ctx, message); err != nil {
		q.log.LogAttrs(ctx, slog.LevelError, "mail send failed",
			slog.String("kind", message.Kind), slog.String("error", err.Error()))
	}
}
