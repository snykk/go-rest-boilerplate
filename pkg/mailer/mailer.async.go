package mailer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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

type jobKind int

const (
	jobOTP jobKind = iota
	jobPasswordReset
)

type otpJob struct {
	kind     jobKind
	payload  string
	receiver string
	// spanCtx carries the originating request's trace identifiers so
	// the worker's send span links back as a child instead of starting
	// an orphan trace. Stored as the immutable SpanContext value so
	// queue dwell time can't trigger context cancellation.
	spanCtx trace.SpanContext
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
func (a *AsyncOTPMailer) SendOTP(ctx context.Context, otpCode, receiver string) error {
	return a.enqueue(otpJob{
		kind:     jobOTP,
		payload:  otpCode,
		receiver: receiver,
		spanCtx:  trace.SpanFromContext(ctx).SpanContext(),
	})
}

// SendPasswordReset enqueues a password-reset email. Same retry +
// backoff guarantees as SendOTP.
func (a *AsyncOTPMailer) SendPasswordReset(ctx context.Context, token, receiver string) error {
	return a.enqueue(otpJob{
		kind:     jobPasswordReset,
		payload:  token,
		receiver: receiver,
		spanCtx:  trace.SpanFromContext(ctx).SpanContext(),
	})
}

func (a *AsyncOTPMailer) enqueue(j otpJob) error {
	select {
	case a.queue <- j:
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
	spanName := "mailer.SendOTP"
	if job.kind == jobPasswordReset {
		spanName = "mailer.SendPasswordReset"
	}
	// Re-attach the originating request's trace as a parent for every
	// retry attempt's span. context.Background as the base avoids
	// inheriting the request's cancellation deadline (the request
	// returns long before the worker runs).
	parentCtx := context.Background()
	if job.spanCtx.IsValid() {
		parentCtx = trace.ContextWithSpanContext(parentCtx, job.spanCtx)
	}

	var lastErr error
	for attempt := 1; attempt <= a.retries; attempt++ {
		ctx, span := observability.Tracer().Start(parentCtx, spanName)
		span.SetAttributes(
			attribute.String("mail.receiver", job.receiver),
			attribute.Int("mail.attempt", attempt),
			attribute.Int("mail.worker", workerID),
		)

		switch job.kind {
		case jobPasswordReset:
			lastErr = a.inner.SendPasswordReset(ctx, job.payload, job.receiver)
		default:
			lastErr = a.inner.SendOTP(ctx, job.payload, job.receiver)
		}
		if lastErr == nil {
			span.SetStatus(codes.Ok, "")
			span.End()
			observability.ObserveMailerOp("sent")
			logger.Info("mail sent", logger.Fields{
				constants.LoggerCategory: constants.LoggerCategoryCache,
				"receiver":               job.receiver,
				"attempt":                attempt,
				"worker":                 workerID,
				"kind":                   spanName,
			})
			return
		}
		span.RecordError(lastErr)
		span.SetStatus(codes.Error, lastErr.Error())
		span.End()

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
	logger.Error("otp email failed after retries", logger.Fields{
		constants.LoggerCategory: constants.LoggerCategoryCache,
		"receiver":               job.receiver,
		"retries":                a.retries,
		"error":                  lastErr.Error(),
	})
}
