// Package audit emits a separate, structured stream of security-
// relevant events (register, login, logout, refresh, OTP verify, etc.)
// independent of the application access log.
//
// The motivation is that operational logs and audit logs have
// different retention, format, and access requirements. Mixing them
// makes after-the-fact forensics expensive: you have to grep through
// 100x as much chatter, and routine log rotation can drop the only
// record of a hostile event.
//
// This package writes JSON lines to its own io.Writer (default os.Stderr,
// override via SetOutput in tests or to file in production). It does
// not depend on the rest of the project — usecases / handlers can call
// audit.Record(...) without creating an import cycle.
package audit

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// EventType enumerates auth-relevant event categories. Names are
// stable strings — log analysis and alerting will key off these.
type EventType string

const (
	EventRegister      EventType = "register"
	EventLoginSuccess  EventType = "login_success"
	EventLoginFailure  EventType = "login_failure"
	EventLogout        EventType = "logout"
	EventRefreshOK     EventType = "refresh_success"
	EventRefreshFail   EventType = "refresh_failure"
	EventOTPSent       EventType = "otp_sent"
	EventOTPVerifyOK   EventType = "otp_verify_success"
	EventOTPVerifyFail EventType = "otp_verify_failure"
	EventOTPLockout    EventType = "otp_lockout"

	EventPasswordChangeOK   EventType = "password_change_success"
	EventPasswordChangeFail EventType = "password_change_failure"
	EventPasswordForgotOK   EventType = "password_forgot_success"
	EventPasswordForgotFail EventType = "password_forgot_failure"
	EventPasswordResetOK    EventType = "password_reset_success"
	EventPasswordResetFail  EventType = "password_reset_failure"
)

// Event is the on-disk shape. Only fields with values are emitted —
// JSON omitempty keeps lines compact.
//
// RequestID + TraceID are correlation IDs: the same pair appears on
// every structured log line for the same HTTP request, so an audit
// entry can be joined back to the application logs (or to spans in
// the tracing backend via TraceID).
type Event struct {
	Time      time.Time `json:"time"`
	Type      EventType `json:"event"`
	Success   bool      `json:"success"`
	UserID    string    `json:"user_id,omitempty"`
	Email     string    `json:"email,omitempty"`
	IP        string    `json:"ip,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	TraceID   string    `json:"trace_id,omitempty"`
	Reason    string    `json:"reason,omitempty"`
}

var (
	mu     sync.Mutex
	out    io.Writer = os.Stderr
	enc    *json.Encoder
	encMu  sync.Mutex
)

func init() {
	enc = json.NewEncoder(out)
}

// SetOutput swaps the destination writer. Safe to call concurrently
// with Record; intended for tests or a startup-time wire-up to
// rotate-aware writers (lumberjack, syslog) in production.
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	out = w
	enc = json.NewEncoder(w)
}

// Record emits one event. Errors writing to the sink are deliberately
// dropped — the only sane fallback when the audit writer fails is
// "don't break the request" — but we expect SetOutput to be wired to
// something durable (file with rotation, journald, etc.) in prod.
func Record(e Event) {
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	encMu.Lock()
	defer encMu.Unlock()
	_ = enc.Encode(e)
}
