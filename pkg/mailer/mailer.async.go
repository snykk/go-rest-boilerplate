package mailer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/observability"
)

// AsyncOTPMailer wraps an OTPMailer so SendOTP enqueues a job and
// returns immediately. The original sync implementation does SMTP IO
// in the request path, making OTP send latency (≈100ms–2s) part of p99
// for /send-otp and /register. Enqueueing lets the HTTP response come
// back on cache/DB latency alone; a worker pool does the delivery and
// retries transient SMTP failures.
//
// ErrQueueFull is returned when the job channel is saturated — the
// caller should decide whether to fail the request or fall back to a
// synchronous send (we log and drop; re-sending is cheap for OTP).
type AsyncOTPMailer struct {
	inner   OTPMailer
	queue   chan otpJob
	workers int
	retries int
	backoff time.Duration

	wg       sync.WaitGroup
	stopOnce sync.Once
	stop     chan struct{}
}

type otpJob struct {
	code     string
	receiver string
}

// ErrQueueFull means the async mailer cannot accept more work without
// blocking; callers should treat this as a soft failure.
var ErrQueueFull = errors.New("otp mailer queue is full")

// NewAsyncOTPMailer starts `workers` goroutines and returns a wrapper
// that implements the OTPMailer interface.
func NewAsyncOTPMailer(inner OTPMailer, workers, queueSize, retries int, backoff time.Duration) *AsyncOTPMailer {
	if workers <= 0 {
		workers = 2
	}
	if queueSize <= 0 {
		queueSize = 64
	}
	if retries <= 0 {
		retries = 3
	}
	if backoff <= 0 {
		backoff = time.Second
	}

	a := &AsyncOTPMailer{
		inner:   inner,
		queue:   make(chan otpJob, queueSize),
		workers: workers,
		retries: retries,
		backoff: backoff,
		stop:    make(chan struct{}),
	}

	for i := 0; i < workers; i++ {
		a.wg.Add(1)
		go a.worker(i)
	}
	return a
}

// SendOTP enqueues the mail job. Returns nil on successful enqueue,
// ErrQueueFull if the channel is saturated.
func (a *AsyncOTPMailer) SendOTP(otpCode string, receiver string) error {
	select {
	case a.queue <- otpJob{code: otpCode, receiver: receiver}:
		return nil
	default:
		return ErrQueueFull
	}
}

// Shutdown signals workers to drain the queue and exit. It blocks until
// all workers finish, or ctx expires.
func (a *AsyncOTPMailer) Shutdown(ctx context.Context) error {
	a.stopOnce.Do(func() {
		close(a.stop)
		// Closing the queue is what actually makes workers return once
		// drained; `stop` just lets in-flight retry sleeps exit fast.
		close(a.queue)
	})

	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *AsyncOTPMailer) worker(id int) {
	defer a.wg.Done()
	for job := range a.queue {
		a.deliver(id, job)
	}
}

func (a *AsyncOTPMailer) deliver(workerID int, job otpJob) {
	var lastErr error
	for attempt := 1; attempt <= a.retries; attempt++ {
		lastErr = a.inner.SendOTP(job.code, job.receiver)
		if lastErr == nil {
			observability.ObserveMailerOp("sent")
			logger.Info("otp email sent", logrus.Fields{
				constants.LoggerCategory: constants.LoggerCategoryCache,
				"receiver":               job.receiver,
				"attempt":                attempt,
				"worker":                 workerID,
			})
			return
		}

		if attempt == a.retries {
			break
		}
		// Exponential backoff with an upper bound so transient 4xx SMTP
		// errors don't wedge a worker for minutes.
		wait := a.backoff * time.Duration(1<<uint(attempt-1))
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}
		select {
		case <-time.After(wait):
		case <-a.stop:
			// Shutdown requested — skip the remaining retries so
			// Shutdown doesn't block on backoff timers.
			return
		}
	}

	observability.ObserveMailerOp("failed")
	logger.Error("otp email failed after retries", logrus.Fields{
		constants.LoggerCategory: constants.LoggerCategoryCache,
		"receiver":               job.receiver,
		"retries":                a.retries,
		"error":                  lastErr.Error(),
	})
}
