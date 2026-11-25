// Package mail writes the identity module's messages, the verification code and the password reset token, as
// plain text with an HTML version, from the templates in templates/, and queues them with the platform mailer.
package mail

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"log/slog"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/mailer"
)

//go:embed templates
var templateFiles embed.FS

// sampleSecret is the code and the token NewMail tries the templates with: each part of a message must show it.
const sampleSecret = "sample-secret-0042"

// Settings are what a Mail writes into its messages besides the secret each one carries.
type Settings struct {
	// Service is the name of the service the messages come from, shown in their header, such as "Blog".
	Service string
	// CodeTTL is how long a verification code lasts.
	CodeTTL time.Duration
	// ResetTTL is how long a password reset token lasts.
	ResetTTL time.Duration
}

// Mail queues messages that carry a secret to a user, each with a short plain subject, a plain-text body and
// an HTML version in a shared layout. It logs only a message it cannot render, by its kind, never its
// recipient or content. It is safe for concurrent use when its queue is.
type Mail struct {
	queue         mailer.Enqueuer
	log           *slog.Logger
	settings      Settings
	verification  message
	passwordReset message
}

var _ usecase.Mail = (*Mail)(nil)

// message is one kind of message: the kind its log records name, its subject and its two templates.
type message struct {
	kind    string
	subject string
	html    *htmltemplate.Template
	text    *texttemplate.Template
}

// content is what the templates of a message are executed with.
type content struct {
	Service  string
	Code     string
	Token    string
	Lifetime string
}

// NewMail returns a Mail that queues its messages with queue, logs to log and writes settings into them. It
// parses the templates once and tries each of them, and returns an error when one is missing, does not parse,
// names a value a message does not have, or leaves out of a part the code or token its message carries.
func NewMail(queue mailer.Enqueuer, log *slog.Logger, settings Settings) (*Mail, error) {
	return newMail(queue, log, settings, templateFiles)
}

func newMail(queue mailer.Enqueuer, log *slog.Logger, settings Settings, files fs.FS) (*Mail, error) {
	verification, err := parseMessage(files, "verification", "identity.verification", "Your verification code")
	if err != nil {
		return nil, err
	}
	passwordReset, err := parseMessage(files, "password_reset", "identity.password_reset", "Reset your password")
	if err != nil {
		return nil, err
	}
	return &Mail{
		queue:         queue,
		log:           log,
		settings:      settings,
		verification:  verification,
		passwordReset: passwordReset,
	}, nil
}

// parseMessage parses the templates of the message named name, templates/<name>.txt and templates/<name>.html
// inside templates/layout.html, and tries them with sample content whose code and token are sampleSecret.
func parseMessage(files fs.FS, name, kind, subject string) (message, error) {
	html, err := htmltemplate.ParseFS(files, "templates/layout.html", "templates/"+name+".html")
	if err != nil {
		return message{}, fmt.Errorf("mail: parse the HTML templates of %s: %w", name, err)
	}
	text, err := texttemplate.ParseFS(files, "templates/"+name+".txt")
	if err != nil {
		return message{}, fmt.Errorf("mail: parse the text template of %s: %w", name, err)
	}

	parsed := message{
		kind:    kind,
		subject: subject,
		html:    html,
		text:    text,
	}
	if err := parsed.try(); err != nil {
		return message{}, fmt.Errorf("mail: try the templates of %s: %w", name, err)
	}
	return parsed, nil
}

// try renders the message with sample content and reports an error unless both parts show the sample secret.
func (m message) try() error {
	sample := content{
		Service:  "Service",
		Code:     sampleSecret,
		Token:    sampleSecret,
		Lifetime: "1 minute",
	}
	text, html, err := m.render(sample)
	if err != nil {
		return err
	}
	if !strings.Contains(text, sampleSecret) || !strings.Contains(html, sampleSecret) {
		return errors.New("a part leaves out the code or token")
	}
	return nil
}

func (m message) render(data content) (text, html string, err error) {
	var textBody, htmlBody bytes.Buffer
	if err := m.text.Execute(&textBody, data); err != nil {
		return "", "", err
	}
	if err := m.html.Execute(&htmlBody, data); err != nil {
		return "", "", err
	}
	return textBody.String(), htmlBody.String(), nil
}

// Verification queues a message of kind identity.verification to email that gives the code and how long it
// lasts.
func (m *Mail) Verification(ctx context.Context, email domain.Email, code string) {
	m.send(ctx, email, m.verification, content{
		Service:  m.settings.Service,
		Code:     code,
		Lifetime: lifetime(m.settings.CodeTTL),
	})
}

// PasswordReset queues a message of kind identity.password_reset to email that gives the token and how long it
// lasts.
func (m *Mail) PasswordReset(ctx context.Context, email domain.Email, token string) {
	m.send(ctx, email, m.passwordReset, content{
		Service:  m.settings.Service,
		Token:    token,
		Lifetime: lifetime(m.settings.ResetTTL),
	})
}

// send queues outgoing to email, written with data. A message it cannot render is not sent; it is logged at
// ERROR with its kind and the error, and nothing of its recipient or content.
func (m *Mail) send(ctx context.Context, email domain.Email, outgoing message, data content) {
	text, html, err := outgoing.render(data)
	if err != nil {
		m.log.LogAttrs(ctx, slog.LevelError, "mail render failed",
			slog.String("kind", outgoing.kind), slog.String("error", err.Error()))
		return
	}

	m.queue.Enqueue(ctx, mailer.Message{
		To:      email.String(),
		Subject: outgoing.subject,
		Text:    text,
		HTML:    html,
		Kind:    outgoing.kind,
	})
}

// lifetime writes ttl in whole hours when it is a whole number of hours, and in minutes, rounded down,
// otherwise: "1 hour", "2 hours", "10 minutes".
func lifetime(ttl time.Duration) string {
	if ttl >= time.Hour && ttl%time.Hour == 0 {
		return plural(int(ttl/time.Hour), "hour")
	}
	return plural(int(ttl/time.Minute), "minute")
}

func plural(count int, unit string) string {
	if count == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", count, unit)
}
