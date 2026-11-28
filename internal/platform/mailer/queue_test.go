package mailer_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/platform/mailer"
)

var errSMTPDown = errors.New("smtp down")

// testSender stores every message it accepts in recorder, refuses its first message when failFirst is set, and
// reports each attempt on attempts.
type testSender struct {
	recorder  *mailer.Recorder
	failFirst bool
	calls     atomic.Int32
	attempts  chan struct{}
}

func (s *testSender) Send(ctx context.Context, message mailer.Message) error {
	defer func() { s.attempts <- struct{}{} }()
	if s.failFirst && s.calls.Add(1) == 1 {
		return errSMTPDown
	}
	return s.recorder.Send(ctx, message)
}

// fixture is a Queue over a testSender that logs JSON into logs.
type fixture struct {
	sender *testSender
	logs   *bytes.Buffer
	queue  *mailer.Queue
}

func newFixture(size int, failFirst bool) *fixture {
	f := &fixture{
		sender: &testSender{
			recorder:  &mailer.Recorder{},
			failFirst: failFirst,
			attempts:  make(chan struct{}, 16),
		},
		logs: &bytes.Buffer{},
	}
	f.queue = mailer.NewQueue(f.sender, slog.New(slog.NewJSONHandler(f.logs, nil)), size, time.Minute)
	return f
}

// run starts the queue in a goroutine and returns the channel its result arrives on.
func (f *fixture) run(ctx context.Context) <-chan error {
	done := make(chan error, 1)
	go func() { done <- f.queue.Run(ctx) }()
	return done
}

// records returns the log records in logs, each with its values as strings.
func records(t *testing.T, logs *bytes.Buffer) []map[string]string {
	t.Helper()
	var records []map[string]string
	for line := range strings.Lines(logs.String()) {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("log line %q is not JSON: %v", line, err)
		}
		values := make(map[string]string, len(record))
		for key, value := range record {
			values[key] = fmt.Sprint(value)
		}
		records = append(records, values)
	}
	return records
}

func messageFixture(number int) mailer.Message {
	return mailer.Message{
		To:      fmt.Sprintf("reader%d@example.com", number),
		Subject: fmt.Sprintf("Subject %d", number),
		Text:    fmt.Sprintf("Body %d", number),
		Kind:    fmt.Sprintf("test.kind%d", number),
	}
}

func awaitDone(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return within 5s")
	}
}

// NewQueue returns a queue that holds between one and MaxQueueSize messages, whatever size it is given: Enqueue
// takes that many and drops the next one as the queue being full.
func TestNewQueue(t *testing.T) {
	tCases := []struct {
		name     string
		size     int
		wantHeld int
	}{
		{name: "a size above MaxQueueSize holds MaxQueueSize messages", size: math.MaxInt, wantHeld: mailer.MaxQueueSize},
		{name: "a size below one holds one message", size: 0, wantHeld: 1},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(tc.size, false)

			for number := range tc.wantHeld + 1 {
				f.queue.Enqueue(context.Background(), messageFixture(number))
			}

			want := map[string]string{"level": "WARN", "msg": "mail queue full", "kind": messageFixture(tc.wantHeld).Kind}
			logs := records(t, f.logs)
			if len(logs) != 1 {
				t.Fatalf("log records = %v, want one matching %v", logs, want)
			}
			for key, value := range want {
				if logs[0][key] != value {
					t.Errorf("log record %s = %q, want %q", key, logs[0][key], value)
				}
			}
		})
	}
}

// Run sends what Enqueue was given in order, drops a message and logs a warning when the queue is full, sends what
// it still holds when its context ends, and logs a failed send without stopping.
func TestQueueRun(t *testing.T) {
	tCases := []struct {
		name              string
		size              int
		failFirst         bool
		runBeforeEnqueue  bool
		messages          []mailer.Message
		sendsBeforeCancel int
		want              []mailer.Message
		wantLogs          []map[string]string
	}{
		{
			name:              "enqueued messages are sent in order",
			size:              10,
			runBeforeEnqueue:  true,
			messages:          []mailer.Message{messageFixture(1), messageFixture(2), messageFixture(3)},
			sendsBeforeCancel: 3,
			want:              []mailer.Message{messageFixture(1), messageFixture(2), messageFixture(3)},
		},
		{
			name:     "a full queue drops and logs the message",
			size:     1,
			messages: []mailer.Message{messageFixture(1), messageFixture(2)},
			want:     []mailer.Message{messageFixture(1)},
			wantLogs: []map[string]string{
				{"level": "WARN", "msg": "mail queue full", "kind": "test.kind2"},
			},
		},
		{
			name:     "messages left when the context ends are still sent",
			size:     10,
			messages: []mailer.Message{messageFixture(1), messageFixture(2), messageFixture(3)},
			want:     []mailer.Message{messageFixture(1), messageFixture(2), messageFixture(3)},
		},
		{
			name:              "a failing sender is logged and the next message still goes",
			size:              10,
			failFirst:         true,
			runBeforeEnqueue:  true,
			messages:          []mailer.Message{messageFixture(1), messageFixture(2)},
			sendsBeforeCancel: 2,
			want:              []mailer.Message{messageFixture(2)},
			wantLogs: []map[string]string{
				{"level": "ERROR", "kind": "test.kind1", "error": "smtp down"},
			},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(tc.size, tc.failFirst)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var done <-chan error
			if tc.runBeforeEnqueue {
				done = f.run(ctx)
			}

			for _, message := range tc.messages {
				f.queue.Enqueue(ctx, message)
			}
			for range tc.sendsBeforeCancel {
				select {
				case <-f.sender.attempts:
				case <-time.After(5 * time.Second):
					t.Fatal("the queue did not send within 5s")
				}
			}
			cancel()
			if done == nil {
				done = f.run(ctx)
			}
			awaitDone(t, done)

			if got := f.sender.recorder.Messages(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("sent messages = %+v, want %+v", got, tc.want)
			}
			logs := records(t, f.logs)
			if len(logs) != len(tc.wantLogs) {
				t.Fatalf("log records = %v, want %d matching %v", logs, len(tc.wantLogs), tc.wantLogs)
			}
			for i, want := range tc.wantLogs {
				for key, value := range want {
					if logs[i][key] != value {
						t.Errorf("log record %d %s = %q, want %q", i, key, logs[i][key], value)
					}
				}
			}
			for _, message := range tc.messages {
				for _, secret := range []string{message.To, message.Subject, message.Text} {
					if strings.Contains(f.logs.String(), secret) {
						t.Errorf("the log contains %q, want no recipient, subject or body", secret)
					}
				}
			}
		})
	}
}

// gateSender holds every send until release is closed and keeps whether its context had ended by then.
type gateSender struct {
	started  chan struct{}
	release  chan struct{}
	recorder *mailer.Recorder
	canceled atomic.Bool
}

func (s *gateSender) Send(ctx context.Context, message mailer.Message) error {
	select {
	case s.started <- struct{}{}:
	default:
	}
	<-s.release
	if ctx.Err() != nil {
		s.canceled.Store(true)
	}
	return s.recorder.Send(ctx, message)
}

// Run finishes the send in flight when its context ends and sends the messages it still holds, all with a
// context that is no longer cancelled, then returns nil.
func TestQueueRunShutdown(t *testing.T) {
	sender := &gateSender{
		started:  make(chan struct{}, 1),
		release:  make(chan struct{}),
		recorder: &mailer.Recorder{},
	}
	queue := mailer.NewQueue(sender, slog.New(slog.DiscardHandler), 10, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- queue.Run(ctx) }()

	queue.Enqueue(ctx, messageFixture(1))
	select {
	case <-sender.started:
	case <-time.After(5 * time.Second):
		t.Fatal("the queue did not start sending within 5s")
	}
	cancel()
	queue.Enqueue(ctx, messageFixture(2))
	close(sender.release)
	awaitDone(t, done)

	want := []mailer.Message{messageFixture(1), messageFixture(2)}
	if got := sender.recorder.Messages(); !reflect.DeepEqual(got, want) {
		t.Errorf("sent messages = %+v, want %+v", got, want)
	}
	if sender.canceled.Load() {
		t.Error("a send ran with a cancelled context, want the shutdown to let it finish")
	}
}

// stuckSender holds every send until the send's context ends, as a mail server that stopped answering does,
// and reports each send it starts on started.
type stuckSender struct {
	started chan struct{}
}

func (s *stuckSender) Send(ctx context.Context, _ mailer.Message) error {
	s.started <- struct{}{}
	<-ctx.Done()
	return ctx.Err()
}

// Run stops sending once its drain timeout has passed since its context ended: it cancels the send in flight,
// drops the messages it still holds and logs how many, so a server that stopped answering cannot hold up a
// shutdown for longer.
func TestQueueRunDrainTimeout(t *testing.T) {
	const drainTimeout = 200 * time.Millisecond
	sender := &stuckSender{started: make(chan struct{}, 3)}
	var logs bytes.Buffer
	queue := mailer.NewQueue(sender, slog.New(slog.NewJSONHandler(&logs, nil)), 10, drainTimeout)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- queue.Run(ctx) }()

	for number := range 3 {
		queue.Enqueue(ctx, messageFixture(number+1))
	}
	select {
	case <-sender.started:
	case <-time.After(5 * time.Second):
		t.Fatal("the queue did not start sending within 5s")
	}
	stopped := time.Now()
	cancel()
	awaitDone(t, done)

	if took := time.Since(stopped); took < drainTimeout || took > drainTimeout+2*time.Second {
		t.Errorf("Run returned %v after its context ended, want %v to %v",
			took, drainTimeout, drainTimeout+2*time.Second)
	}
	want := []map[string]string{
		{"level": "ERROR", "msg": "mail send failed", "kind": "test.kind1", "error": "context canceled"},
		{"level": "WARN", "msg": "mail queue stopped with messages unsent", "unsent": "2"},
	}
	got := records(t, &logs)
	if len(got) != len(want) {
		t.Fatalf("log records = %v, want %d matching %v", got, len(want), want)
	}
	for i, wantRecord := range want {
		for key, value := range wantRecord {
			if got[i][key] != value {
				t.Errorf("log record %d %s = %q, want %q", i, key, got[i][key], value)
			}
		}
	}
}

// Enqueue drops a message given once Run has returned, since nothing would send it, and logs a warning with its
// Kind.
func TestQueueEnqueueAfterRun(t *testing.T) {
	sender := &mailer.Recorder{}
	var logs bytes.Buffer
	queue := mailer.NewQueue(sender, slog.New(slog.NewJSONHandler(&logs, nil)), 10, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := queue.Run(ctx); err != nil {
		t.Fatalf("Run = %v, want nil", err)
	}

	queue.Enqueue(context.Background(), messageFixture(1))

	if got := sender.Messages(); len(got) != 0 {
		t.Errorf("sent messages = %+v, want none", got)
	}
	want := map[string]string{"level": "WARN", "msg": "mail queue stopped, message dropped", "kind": "test.kind1"}
	got := records(t, &logs)
	if len(got) != 1 {
		t.Fatalf("log records = %v, want one matching %v", got, want)
	}
	for key, value := range want {
		if got[0][key] != value {
			t.Errorf("log record %s = %q, want %q", key, got[0][key], value)
		}
	}
}

// Run and Enqueue account for every message while the queue stops: each message enqueued before Run returns,
// while it returns or after, is either sent or counted in a log record, and none is lost without a trace.
func TestQueueRunAccountsForEveryMessage(t *testing.T) {
	const enqueuers, perEnqueuer, afterRun = 8, 50, 5
	sender := &mailer.Recorder{}
	var logs bytes.Buffer
	queue := mailer.NewQueue(sender, slog.New(slog.NewJSONHandler(&logs, nil)), 16, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- queue.Run(ctx) }()

	var enqueuing sync.WaitGroup
	for range enqueuers {
		enqueuing.Go(func() {
			for number := range perEnqueuer {
				queue.Enqueue(context.Background(), messageFixture(number))
			}
		})
	}
	cancel()
	awaitDone(t, done)
	enqueuing.Wait()
	for number := range afterRun {
		queue.Enqueue(context.Background(), messageFixture(number))
	}

	reported := 0
	for _, record := range records(t, &logs) {
		switch record["msg"] {
		case "mail queue full", "mail queue stopped, message dropped":
			reported++
		case "mail queue stopped with messages unsent":
			unsent, err := strconv.Atoi(record["unsent"])
			if err != nil {
				t.Fatalf("log record unsent = %q, want a count", record["unsent"])
			}
			reported += unsent
		}
	}
	if sent, want := len(sender.Messages()), enqueuers*perEnqueuer+afterRun; sent+reported != want {
		t.Errorf("messages sent = %d, reported = %d; want %d in all", sent, reported, want)
	}
}
