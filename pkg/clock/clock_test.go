package clock_test

import (
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/pkg/clock"
	"github.com/stretchr/testify/assert"
)

func TestRealClock_NowIsRecent(t *testing.T) {
	got := clock.RealClock{}.Now()
	assert.WithinDuration(t, time.Now(), got, time.Second)
}

func TestFrozen_AlwaysReturnsSameInstant(t *testing.T) {
	at := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	c := clock.Frozen(at)

	first := c.Now()
	time.Sleep(2 * time.Millisecond)
	second := c.Now()

	assert.Equal(t, at, first)
	assert.Equal(t, at, second, "Frozen must not advance between calls")
}

func TestStub_AdvanceMovesNowForward(t *testing.T) {
	at := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	s := &clock.Stub{}
	s.Set(at)

	assert.Equal(t, at, s.Now())
	s.Advance(time.Hour)
	assert.Equal(t, at.Add(time.Hour), s.Now())
}
