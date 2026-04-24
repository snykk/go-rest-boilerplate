package mailer_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/pkg/mailer"
	"github.com/stretchr/testify/assert"
)

// stubMailer records every SendOTP call and can be configured to fail
// the first N attempts to exercise the retry path.
type stubMailer struct {
	mu          sync.Mutex
	calls       int32
	failUntil   int32
	forceErr    error
	deliveredTo []string
}

func (s *stubMailer) SendOTP(code, receiver string) error {
	n := atomic.AddInt32(&s.calls, 1)
	if s.forceErr != nil && n <= s.failUntil {
		return s.forceErr
	}
	s.mu.Lock()
	s.deliveredTo = append(s.deliveredTo, receiver)
	s.mu.Unlock()
	return nil
}

func TestAsyncMailer_Delivers(t *testing.T) {
	stub := &stubMailer{}
	async := mailer.NewAsyncOTPMailer(stub, 1, 4, 3, 10*time.Millisecond)

	assert.NoError(t, async.SendOTP("123456", "alice@example.com"))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	assert.NoError(t, async.Shutdown(ctx))

	stub.mu.Lock()
	defer stub.mu.Unlock()
	assert.Equal(t, []string{"alice@example.com"}, stub.deliveredTo)
}

func TestAsyncMailer_RetriesTransientFailures(t *testing.T) {
	stub := &stubMailer{forceErr: errors.New("smtp timeout"), failUntil: 2}
	async := mailer.NewAsyncOTPMailer(stub, 1, 4, 3, 10*time.Millisecond)

	assert.NoError(t, async.SendOTP("123456", "bob@example.com"))

	// Wait until the worker has made all three attempts before shutting
	// down — Shutdown cancels pending retry backoffs, which would race
	// with delivery otherwise.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && atomic.LoadInt32(&stub.calls) < 3 {
		time.Sleep(5 * time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	assert.NoError(t, async.Shutdown(ctx))

	assert.Equal(t, int32(3), atomic.LoadInt32(&stub.calls), "should retry twice after initial failure")
	stub.mu.Lock()
	defer stub.mu.Unlock()
	assert.Equal(t, []string{"bob@example.com"}, stub.deliveredTo)
}

func TestAsyncMailer_ErrQueueFull(t *testing.T) {
	// zero workers + tiny queue + a mailer that blocks forever → queue fills.
	blocker := make(chan struct{})
	stub := &blockingMailer{release: blocker}
	async := mailer.NewAsyncOTPMailer(stub, 1, 1, 1, time.Millisecond)

	// First send occupies the worker; second fills the queue; third must fail.
	_ = async.SendOTP("1", "a@a")
	_ = async.SendOTP("2", "b@b")
	// Give the worker a beat to pick up the first job.
	time.Sleep(20 * time.Millisecond)
	_ = async.SendOTP("3", "c@c")
	err := async.SendOTP("4", "d@d")
	assert.ErrorIs(t, err, mailer.ErrQueueFull)

	close(blocker)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = async.Shutdown(ctx)
}

type blockingMailer struct {
	release chan struct{}
}

func (b *blockingMailer) SendOTP(code, receiver string) error {
	<-b.release
	return nil
}
