package audit_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/pkg/audit"
	"github.com/stretchr/testify/assert"
)

func TestRecord_EmitsJSONLine(t *testing.T) {
	var buf bytes.Buffer
	audit.SetOutput(&buf)

	audit.Record(audit.Event{
		Type:      audit.EventLoginSuccess,
		Success:   true,
		Email:     "alice@example.com",
		UserID:    "u-1",
		IP:        "10.0.0.1",
		RequestID: "req-abc",
	})

	line := strings.TrimSpace(buf.String())
	assert.NotEmpty(t, line)

	var got audit.Event
	assert.NoError(t, json.Unmarshal([]byte(line), &got))
	assert.Equal(t, audit.EventLoginSuccess, got.Type)
	assert.True(t, got.Success)
	assert.Equal(t, "alice@example.com", got.Email)
	assert.Equal(t, "req-abc", got.RequestID)
	// Time is auto-filled when not provided.
	assert.False(t, got.Time.IsZero())
	assert.WithinDuration(t, time.Now(), got.Time, 5*time.Second)
}

func TestRecord_OmitsEmptyFields(t *testing.T) {
	var buf bytes.Buffer
	audit.SetOutput(&buf)

	audit.Record(audit.Event{
		Type:    audit.EventLoginFailure,
		Success: false,
		Reason:  "wrong password",
	})

	// UserID, Email, IP not set → must not appear in the JSON line.
	line := buf.String()
	assert.NotContains(t, line, `"user_id"`)
	assert.NotContains(t, line, `"email"`)
	assert.NotContains(t, line, `"ip"`)
	assert.Contains(t, line, `"reason":"wrong password"`)
}
