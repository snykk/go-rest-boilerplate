package clock_test

import (
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/platform/clock"
)

// Now answers in UTC, in whole microseconds.
func TestRealNow(t *testing.T) {
	for range 5 { // a single reading can land on a whole microsecond by chance
		now := (clock.Real{}).Now()
		if location := now.Location(); location != time.UTC {
			t.Fatalf("Now location = %v, want UTC", location)
		}
		if nanoseconds := now.Nanosecond() % 1000; nanoseconds != 0 {
			t.Fatalf("Now = %v, want whole microseconds (%d ns left over)", now, nanoseconds)
		}
	}
}
