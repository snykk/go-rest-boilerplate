package mail_test

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/mail"
	"github.com/snykk/go-rest-boilerplate/internal/platform/mailer"
)

const service = "Standard Blog"

// Verification queues one message of kind identity.verification to the address, with a short plain subject,
// whose text and HTML both give the code and how long it lasts, the HTML inside the layout that names the
// service.
func TestMailVerification(t *testing.T) {
	tCases := []struct {
		name         string
		codeTTL      time.Duration
		wantLifetime string
	}{
		{name: "a code that lasts ten minutes", codeTTL: 10 * time.Minute, wantLifetime: "in 10 minutes."},
		{name: "a code that lasts one minute", codeTTL: time.Minute, wantLifetime: "in 1 minute."},
		{name: "a code that lasts an hour", codeTTL: time.Hour, wantLifetime: "in 1 hour."},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			var queue mailer.Recorder

			mailFixture(t, &queue, tc.codeTTL, time.Hour).Verification(t.Context(), address(t), "123456")

			got := oneMessage(t, queue.Messages(), "identity.verification")
			assertContains(t, "text", got.Text, "123456", tc.wantLifetime)
			assertContains(t, "HTML", got.HTML, "123456", tc.wantLifetime, service, "<table")
		})
	}
}

// PasswordReset queues one message of kind identity.password_reset to the address, with a short plain
// subject, whose text and HTML both give the token and how long it lasts, the HTML inside the layout that names
// the service. Markup in the token is escaped in the HTML and kept as it is in the text.
func TestMailPasswordReset(t *testing.T) {
	tCases := []struct {
		name       string
		resetTTL   time.Duration
		token      string
		wantInText []string
		wantInHTML []string
	}{
		{
			name:       "a token that lasts thirty minutes",
			resetTTL:   30 * time.Minute,
			token:      "reset-token",
			wantInText: []string{"reset-token", "in 30 minutes."},
			wantInHTML: []string{"reset-token", "in 30 minutes.", service, "<table"},
		},
		{
			name:       "a token that lasts two hours",
			resetTTL:   2 * time.Hour,
			token:      "reset-token",
			wantInText: []string{"reset-token", "in 2 hours."},
			wantInHTML: []string{"reset-token", "in 2 hours."},
		},
		{
			name:       "markup in a token is escaped in the HTML",
			resetTTL:   30 * time.Minute,
			token:      "<script>alert(1)</script>",
			wantInText: []string{"<script>alert(1)</script>"},
			wantInHTML: []string{"&lt;script&gt;alert(1)&lt;/script&gt;"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			var queue mailer.Recorder

			mailFixture(t, &queue, time.Minute, tc.resetTTL).PasswordReset(t.Context(), address(t), tc.token)

			got := oneMessage(t, queue.Messages(), "identity.password_reset")
			assertContains(t, "text", got.Text, tc.wantInText...)
			assertContains(t, "HTML", got.HTML, tc.wantInHTML...)
			if strings.Contains(got.HTML, "<script") {
				t.Errorf("message HTML = %q, want no script element", got.HTML)
			}
		})
	}
}

// mailFixture is a Mail for the service that queues with queue, failing the test when there is none.
func mailFixture(t *testing.T, queue mailer.Enqueuer, codeTTL, resetTTL time.Duration) *mail.Mail {
	t.Helper()
	got, err := mail.NewMail(queue, slog.New(slog.DiscardHandler), mail.Settings{
		Service:  service,
		CodeTTL:  codeTTL,
		ResetTTL: resetTTL,
	})
	if err != nil {
		t.Fatalf("NewMail = %v", err)
	}
	return got
}

// address is ada@example.com as a domain.Email.
func address(t *testing.T) domain.Email {
	t.Helper()
	email, err := domain.NewEmail("ada@example.com")
	if err != nil {
		t.Fatalf("NewEmail = %v", err)
	}
	return email
}

// oneMessage checks that messages hold one message of kind to ada@example.com, with a subject on one line,
// and returns it.
func oneMessage(t *testing.T, messages []mailer.Message, kind string) mailer.Message {
	t.Helper()
	if len(messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(messages))
	}
	got := messages[0]
	if got.Kind != kind || got.To != "ada@example.com" || got.Subject == "" || strings.Contains(got.Subject, "\n") {
		t.Errorf("message = %s to %s with subject %q, want %s to ada@example.com with a one-line subject",
			got.Kind, got.To, got.Subject, kind)
	}
	return got
}

// assertContains checks that body, the part of a message named part, contains each of wants.
func assertContains(t *testing.T, part, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("message %s = %q, want it to contain %q", part, body, want)
		}
	}
}
