package mailer

import (
	"net"
	"net/mail"
	"strconv"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
)

// MaxQueueSize is the most messages a Queue holds, and the largest MAIL_QUEUE_SIZE that LoadConfig accepts.
const MaxQueueSize = 10_000

// Config holds the mailer's settings, read from MAIL_* variables.
type Config struct {
	// Addr is the SMTP server's host and port.
	Addr string
	// From is the sender of every message: an address, with an optional display name.
	From string
	// Username authenticates with PLAIN, together with Password, when it is set. PLAIN is only sent over a
	// connection that is encrypted or goes to the local host.
	Username string
	// Password is Username's password. It never prints: fmt, log/slog and JSON show it as [REDACTED].
	Password config.Secret
	// StartTLS upgrades the connection with STARTTLS before anything else is sent, and fails when the server
	// does not offer it.
	StartTLS bool
	// Timeout bounds one delivery as a whole: connecting, the conversation and the transfer.
	Timeout time.Duration
	// QueueSize is the number of messages a Queue holds before it drops new ones, from 1 to MaxQueueSize.
	QueueSize int
}

// LoadConfig reads MAIL_SMTP_ADDR (default localhost:1025), MAIL_FROM (no-reply@example.com),
// MAIL_SMTP_USERNAME and MAIL_SMTP_PASSWORD (empty), MAIL_SMTP_STARTTLS (false), MAIL_TIMEOUT (10s) and
// MAIL_QUEUE_SIZE (100, at most MaxQueueSize) from source. It reports every invalid setting in one error,
// including an address with no port, a queue size above MaxQueueSize, and a sender that is not an email
// address or whose local part needs quotes, which SMTP.Send refuses.
func LoadConfig(source config.Source) (Config, error) {
	reader := config.NewReader(source)
	settings := Config{
		Addr:      reader.String("MAIL_SMTP_ADDR", "localhost:1025"),
		From:      reader.String("MAIL_FROM", "no-reply@example.com"),
		Username:  reader.String("MAIL_SMTP_USERNAME", ""),
		Password:  config.NewSecret(reader.String("MAIL_SMTP_PASSWORD", "")),
		StartTLS:  reader.Bool("MAIL_SMTP_STARTTLS", false),
		Timeout:   reader.Duration("MAIL_TIMEOUT", 10*time.Second),
		QueueSize: int(reader.PositiveInt("MAIL_QUEUE_SIZE", 100)),
	}
	if _, _, err := net.SplitHostPort(settings.Addr); err != nil {
		reader.Fail("MAIL_SMTP_ADDR", "must be a host and a port such as localhost:1025")
	}
	if from, err := mail.ParseAddress(settings.From); err != nil {
		reader.Fail("MAIL_FROM", "must be an email address such as no-reply@example.com")
	} else if checkAddress(from) != nil {
		reader.Fail("MAIL_FROM", "must be an email address whose local part needs no quotes, such as no-reply@example.com")
	}
	if settings.QueueSize > MaxQueueSize {
		reader.Fail("MAIL_QUEUE_SIZE", "must be at most "+strconv.Itoa(MaxQueueSize))
	}

	if err := reader.Err(); err != nil {
		return Config{}, err
	}
	return settings, nil
}
