package mailer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// SMTP is a Sender that delivers each message over a connection of its own to the server in its Config. It is
// safe for concurrent use.
type SMTP struct {
	config Config
}

// NewSMTP returns an SMTP sender for config, whose Addr, From and Timeout must be set; LoadConfig guarantees
// them.
func NewSMTP(config Config) *SMTP {
	return &SMTP{config: config}
}

// Send delivers message from the configured sender as a text/plain message in UTF-8 or, when it has HTML, as
// a multipart/alternative message whose text/plain part comes before its text/html part, both in UTF-8 and
// quoted-printable, between boundaries drawn at random. It connects, upgrades the connection with STARTTLS
// when the Config asks for it, authenticates with PLAIN when it has a username, and sends. The whole delivery
// is bounded by the Config's Timeout and by ctx.
//
// Before it connects, Send refuses a sender or a recipient that is not an email address, or whose address
// could end the angle brackets of the SMTP envelope or add a parameter to it: an address that holds
// whitespace, a control character, < or >, or whose local part needs quotes, such as
// "x> NOTIFY=SUCCESS"@example.com. Addresses outside ASCII, such as josé@example.com, and a quoted display
// name, such as "Mail Team" <no-reply@example.com>, are accepted. It also returns an error when the server
// cannot be reached or refuses a step, or when STARTTLS is required and not offered. The error says whether
// the sender or the recipient was refused, or names the step and the server's reply code; it never repeats an
// address, the subject or a body.
func (s *SMTP) Send(ctx context.Context, message Message) error {
	from, err := parseAddress(s.config.From, "sender")
	if err != nil {
		return err
	}
	to, err := parseAddress(message.To, "recipient")
	if err != nil {
		return err
	}
	body, err := render(from, to, message, time.Now())
	if err != nil {
		return err
	}
	host, _, err := net.SplitHostPort(s.config.Addr)
	if err != nil {
		return errors.New("mailer: the server address is not a host and a port")
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", s.config.Addr)
	if err != nil {
		return fmt.Errorf("mailer: connect: %w", err)
	}
	defer func() { _ = connection.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}
	stop := context.AfterFunc(ctx, func() { _ = connection.Close() })
	defer stop()

	if err := s.converse(connection, host, from.Address, to.Address, body); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %w", ctx.Err(), err)
		}
		return err
	}
	return nil
}

// parseAddress parses text as the address of role, the sender or the recipient, and refuses one that
// checkAddress refuses. The error names role, never the address.
func parseAddress(text, role string) (*mail.Address, error) {
	address, err := mail.ParseAddress(text)
	if err != nil {
		return nil, fmt.Errorf("mailer: the %s is not an email address", role)
	}
	if err := checkAddress(address); err != nil {
		return nil, fmt.Errorf("mailer: the %s cannot go in an SMTP envelope: %w", role, err)
	}
	return address, nil
}

// checkAddress returns an error when the address could end the angle brackets of an SMTP envelope path or
// add a parameter after them, as "x> NOTIFY=SUCCESS"@example.com does once mail.ParseAddress has unquoted
// it: when it holds whitespace, a control character, < or >, or when its local part, before the last @,
// holds a character that is neither RFC 5322 atext, a dot nor outside ASCII, which only quotes allow. The
// error never repeats the address.
func checkAddress(address *mail.Address) error {
	if strings.ContainsFunc(address.Address, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r) || r == '<' || r == '>'
	}) {
		return errors.New("the address holds whitespace, a control character, < or >")
	}

	local := address.Address
	if at := strings.LastIndexByte(local, '@'); at >= 0 {
		local = local[:at]
	}
	if strings.ContainsFunc(local, needsQuotes) {
		return errors.New("the local part of the address needs quotes")
	}
	return nil
}

// needsQuotes reports whether r may stand in the local part of an address only between quotes: it is neither
// RFC 5322 atext, a dot, nor a character outside ASCII (RFC 6532).
func needsQuotes(r rune) bool {
	if r >= utf8.RuneSelf || 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9' {
		return false
	}
	return !strings.ContainsRune("!#$%&'*+-/=?^_`{|}~.", r)
}

func (s *SMTP) converse(connection net.Conn, host, from, to string, body []byte) error {
	client, err := smtp.NewClient(connection, host)
	if err != nil {
		return stepError("greeting", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Hello("localhost"); err != nil {
		return stepError("hello", err)
	}
	if s.config.StartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("mailer: the server does not offer STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return stepError("starttls", err)
		}
	}
	if s.config.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.config.Username, s.config.Password.Reveal(), host)); err != nil {
			return stepError("auth", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return stepError("sender", err)
	}
	if err := client.Rcpt(to); err != nil {
		return stepError("recipient", err)
	}
	data, err := client.Data()
	if err != nil {
		return stepError("data", err)
	}
	if _, err := data.Write(body); err != nil {
		return stepError("data", err)
	}
	if err := data.Close(); err != nil {
		return stepError("data", err)
	}

	_ = client.Quit()
	return nil
}

// stepError names the step that failed. The text of a server's reply is dropped and only its code kept,
// because servers echo the recipient's address in it.
func stepError(step string, err error) error {
	if reply, ok := errors.AsType[*textproto.Error](err); ok {
		return fmt.Errorf("mailer: %s: the server replied %d", step, reply.Code)
	}
	return fmt.Errorf("mailer: %s: %w", step, err)
}

func render(from, to *mail.Address, message Message, now time.Time) ([]byte, error) {
	fromHeader, toHeader := from.String(), to.String()
	if strings.ContainsAny(fromHeader+toHeader, "\r\n") {
		return nil, errors.New("mailer: an address contains a line break")
	}
	_, domain, _ := strings.Cut(from.Address, "@")

	var rendered bytes.Buffer
	writeHeaders(&rendered, [][2]string{
		{"From", fromHeader},
		{"To", toHeader},
		{"Subject", mime.QEncoding.Encode("utf-8", message.Subject)},
		{"Date", now.UTC().Format(time.RFC1123Z)},
		{"Message-ID", "<" + rand.Text() + "@" + domain + ">"},
		{"MIME-Version", "1.0"},
	})

	if message.HTML == "" {
		writeHeaders(&rendered, [][2]string{
			{"Content-Type", textPlain},
			{"Content-Transfer-Encoding", "quoted-printable"},
		})
		rendered.WriteString("\r\n")
		if err := writeQuotedPrintable(&rendered, message.Text); err != nil {
			return nil, err
		}
		return rendered.Bytes(), nil
	}

	if err := writeAlternative(&rendered, message.Text, message.HTML); err != nil {
		return nil, err
	}
	return rendered.Bytes(), nil
}

const (
	textPlain = "text/plain; charset=utf-8"
	textHTML  = "text/html; charset=utf-8"
)

func writeHeaders(rendered *bytes.Buffer, headers [][2]string) {
	for _, header := range headers {
		fmt.Fprintf(rendered, "%s: %s\r\n", header[0], header[1])
	}
}

// writeAlternative writes the Content-Type header and the body of a multipart/alternative message: text as
// its first part and html as its second, each quoted-printable.
func writeAlternative(rendered *bytes.Buffer, text, html string) error {
	parts := multipart.NewWriter(rendered)
	contentType := mime.FormatMediaType("multipart/alternative", map[string]string{"boundary": parts.Boundary()})
	writeHeaders(rendered, [][2]string{{"Content-Type", contentType}})
	rendered.WriteString("\r\n")

	for _, version := range []struct{ contentType, body string }{{textPlain, text}, {textHTML, html}} {
		part, err := parts.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {version.contentType},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return fmt.Errorf("mailer: start a part: %w", err)
		}
		if err := writeQuotedPrintable(part, version.body); err != nil {
			return err
		}
	}
	if err := parts.Close(); err != nil {
		return fmt.Errorf("mailer: close the parts: %w", err)
	}
	return nil
}

func writeQuotedPrintable(destination io.Writer, body string) error {
	encoder := quotedprintable.NewWriter(destination)
	if _, err := encoder.Write([]byte(body)); err != nil {
		return fmt.Errorf("mailer: encode the body: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("mailer: encode the body: %w", err)
	}
	return nil
}
