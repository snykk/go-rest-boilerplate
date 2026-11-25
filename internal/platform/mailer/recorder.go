package mailer

import (
	"context"
	"slices"
	"sync"
)

// Recorder keeps every message it is given in memory, in order, and delivers none. It is both a Sender and an
// Enqueuer, completing each call before it returns. The zero value is ready to use, and a Recorder is safe for
// concurrent use.
type Recorder struct {
	mu       sync.Mutex
	messages []Message
}

// Send stores message and returns nil.
func (r *Recorder) Send(_ context.Context, message Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = append(r.messages, message)
	return nil
}

// Enqueue stores message.
func (r *Recorder) Enqueue(ctx context.Context, message Message) {
	_ = r.Send(ctx, message)
}

// Messages returns a copy of the messages stored so far, oldest first.
func (r *Recorder) Messages() []Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.messages)
}
