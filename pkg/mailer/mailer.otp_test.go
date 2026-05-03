package mailer

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRenderOTPBody_ContainsCodeAndYear(t *testing.T) {
	body, err := renderOTPBody("987654")
	assert.NoError(t, err)
	assert.Contains(t, body, "987654", "code must appear in the rendered HTML")
	assert.Contains(t, body, strconv.Itoa(time.Now().Year()), "current year must be embedded")
	assert.Contains(t, body, defaultAppName)
}

func TestRenderOTPBody_AutoEscapesUnsafeInput(t *testing.T) {
	// html/template should escape <script>; if someone ever wires up
	// user-controlled data into the OTP code field this guards it.
	body, err := renderOTPBody("<script>alert(1)</script>")
	assert.NoError(t, err)
	assert.NotContains(t, body, "<script>alert(1)</script>")
	assert.True(t, strings.Contains(body, "&lt;script&gt;"))
}
