// Package clock provides the current time.
package clock

import "time"

// Real reads the system clock.
type Real struct{}

// Now returns the current time in UTC, truncated to whole microseconds, the precision PostgreSQL keeps, so a
// stored time reads back equal to the one written. It carries no monotonic clock reading.
func (Real) Now() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}
