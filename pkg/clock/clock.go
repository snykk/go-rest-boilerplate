// Package clock abstracts time.Now() so code that depends on the
// current time (token expiry, OTP TTL, audit timestamps) can be
// exercised deterministically in tests.
//
// Production code should depend on the Clock interface rather than
// reaching for time.Now() directly. The default implementation,
// RealClock, calls time.Now and is the value injected by the wiring
// in cmd/api. Tests substitute Frozen or Stub to control the clock
// without sleeping.
package clock

import "time"

// Clock returns the current wall-clock time. Mockable.
type Clock interface {
	Now() time.Time
}

// RealClock delegates to time.Now. Use this in production wiring.
type RealClock struct{}

// Now satisfies Clock.
func (RealClock) Now() time.Time { return time.Now() }

// Frozen returns a Clock that always reports the same instant.
// Useful for tests that assert exact expiry timestamps without
// flake from clock advancement between calls.
func Frozen(t time.Time) Clock { return frozen{t} }

type frozen struct{ t time.Time }

func (f frozen) Now() time.Time { return f.t }

// Stub is a hand-controlled Clock that lets tests advance time
// step-by-step (e.g., to drive token expiry without actually
// sleeping). The zero value reports time.Time{}; call Set to seed
// it before use.
type Stub struct{ t time.Time }

// Now satisfies Clock.
func (s *Stub) Now() time.Time { return s.t }

// Set replaces the reported instant.
func (s *Stub) Set(t time.Time) { s.t = t }

// Advance moves the reported instant forward by d.
func (s *Stub) Advance(d time.Duration) { s.t = s.t.Add(d) }
