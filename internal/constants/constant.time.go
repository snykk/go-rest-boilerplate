package constants

import "time"

// GMT7 is the project's canonical display/local timezone (WIB / UTC+7).
// Use it for human-facing timestamps (logs, seed data, formatted
// responses). Domain types intentionally do NOT depend on this — the
// domain stores UTC; converting to GMT7 is a presentation concern.
var GMT7 = time.FixedZone("GMT+7", 7*60*60)
