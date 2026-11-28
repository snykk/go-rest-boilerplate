package mailer_test

import (
	"cmp"
	"context"
	"reflect"
	"slices"
	"sync"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/mailer"
)

// Recorder keeps every message given to Send or Enqueue and returns a copy of them, oldest first, from Messages.
// It is safe for concurrent use.
func TestRecorder(t *testing.T) {
	tCases := []struct {
		name       string
		use        func(t *testing.T, recorder *mailer.Recorder)
		want       []mailer.Message
		sortByKind bool
	}{
		{
			name: "messages come back oldest first",
			use: func(t *testing.T, recorder *mailer.Recorder) {
				send(t, recorder, messageFixture(1))
				recorder.Enqueue(context.Background(), messageFixture(2))
				send(t, recorder, messageFixture(3))
			},
			want: []mailer.Message{messageFixture(1), messageFixture(2), messageFixture(3)},
		},
		{
			name: "the returned messages are a copy",
			use: func(t *testing.T, recorder *mailer.Recorder) {
				send(t, recorder, messageFixture(1))
				send(t, recorder, messageFixture(2))

				returned := recorder.Messages()
				returned[0] = messageFixture(3)
			},
			want: []mailer.Message{messageFixture(1), messageFixture(2)},
		},
		{
			name: "concurrent sends and enqueues are all kept",
			use: func(t *testing.T, recorder *mailer.Recorder) {
				var calls sync.WaitGroup
				for number := range 100 {
					calls.Go(func() {
						if number%2 == 0 {
							send(t, recorder, messageFixture(number))
						} else {
							recorder.Enqueue(context.Background(), messageFixture(number))
						}
						_ = recorder.Messages()
					})
				}
				calls.Wait()
			},
			want:       messagesUpTo(100),
			sortByKind: true,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &mailer.Recorder{}

			tc.use(t, recorder)

			got, want := recorder.Messages(), tc.want
			if tc.sortByKind {
				got, want = byKind(got), byKind(want)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Messages = %+v, want %+v", got, want)
			}
		})
	}
}

func send(t *testing.T, recorder *mailer.Recorder, message mailer.Message) {
	t.Helper()
	if err := recorder.Send(context.Background(), message); err != nil {
		t.Errorf("Send = %v, want nil", err)
	}
}

// messagesUpTo returns messageFixture of every number from 0 to n-1.
func messagesUpTo(n int) []mailer.Message {
	messages := make([]mailer.Message, 0, n)
	for number := range n {
		messages = append(messages, messageFixture(number))
	}
	return messages
}

// byKind returns a copy of messages sorted by Kind.
func byKind(messages []mailer.Message) []mailer.Message {
	return slices.SortedFunc(slices.Values(messages), func(a, b mailer.Message) int { return cmp.Compare(a.Kind, b.Kind) })
}
