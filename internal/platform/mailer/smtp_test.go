package mailer_test

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/textproto"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/platform/mailer"
)

// envelope is what a test SMTP server received in one session.
type envelope struct {
	from, to, data, user, password string
}

type smtpOptions struct {
	advertiseAuth bool
	refuseRcpt    bool
	stall         bool
	closed        bool // nothing listens on the address
}

// smtpServer is an SMTP server on a local port that keeps the envelope of the first message it accepts.
type smtpServer struct {
	address string
	got     chan envelope
}

func startSMTPServer(t *testing.T, options smtpOptions) *smtpServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen = %v", err)
	}
	t.Cleanup(func() { listener.Close() })
	server := &smtpServer{address: listener.Addr().String(), got: make(chan envelope, 1)}
	if options.closed {
		listener.Close()
		return server
	}
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go server.serve(connection, options)
		}
	}()
	return server
}

// angle returns what the first pair of angle brackets in text encloses.
func angle(text string) string {
	_, rest, _ := strings.Cut(text, "<")
	inner, _, _ := strings.Cut(rest, ">")
	return inner
}

func (s *smtpServer) serve(connection net.Conn, options smtpOptions) {
	protocol := textproto.NewConn(connection)
	defer protocol.Close()
	if options.stall {
		_, _ = protocol.ReadLine()
		return
	}

	var received envelope
	_ = protocol.PrintfLine("220 localhost ESMTP")
	for {
		line, err := protocol.ReadLine()
		if err != nil {
			return
		}
		verb, argument, _ := strings.Cut(line, " ")
		switch strings.ToUpper(verb) {
		case "EHLO":
			if options.advertiseAuth {
				_ = protocol.PrintfLine("250-localhost")
				_ = protocol.PrintfLine("250 AUTH PLAIN")
			} else {
				_ = protocol.PrintfLine("250 localhost")
			}
		case "AUTH":
			_, payload, _ := strings.Cut(argument, " ")
			credentials, _ := base64.StdEncoding.DecodeString(payload)
			if parts := strings.Split(string(credentials), "\x00"); len(parts) == 3 {
				received.user, received.password = parts[1], parts[2]
			}
			_ = protocol.PrintfLine("235 2.7.0 authenticated")
		case "MAIL":
			received.from = angle(argument)
			_ = protocol.PrintfLine("250 ok")
		case "RCPT":
			if options.refuseRcpt {
				_ = protocol.PrintfLine("550 5.1.1 <%s> user unknown", angle(argument))
				continue
			}
			received.to = angle(argument)
			_ = protocol.PrintfLine("250 ok")
		case "DATA":
			_ = protocol.PrintfLine("354 go ahead")
			data, _ := io.ReadAll(protocol.DotReader())
			received.data = string(data)
			_ = protocol.PrintfLine("250 ok")
			select {
			case s.got <- received:
			default:
			}
		case "QUIT":
			_ = protocol.PrintfLine("221 bye")
			return
		default:
			_ = protocol.PrintfLine("502 not implemented")
		}
	}
}

func (s *smtpServer) envelope(t *testing.T) envelope {
	t.Helper()
	select {
	case received := <-s.got:
		return received
	case <-time.After(5 * time.Second):
		t.Fatal("the server received no message within 5s")
		return envelope{}
	}
}

// parsed is a received message split into its headers and its decoded body.
type parsed struct {
	header mail.Header
	body   string
}

func parseMessage(t *testing.T, data string) parsed {
	t.Helper()
	message, err := mail.ReadMessage(strings.NewReader(data))
	if err != nil {
		t.Fatalf("mail.ReadMessage = %v", err)
	}
	body, err := io.ReadAll(quotedprintable.NewReader(message.Body))
	if err != nil {
		t.Fatalf("reading the body = %v", err)
	}
	return parsed{header: message.Header, body: strings.ReplaceAll(string(body), "\r\n", "\n")}
}

// htmlFixture is an HTML body with a character outside ASCII and a line longer than quoted-printable allows.
var htmlFixture = "<p>Hello Zoë,</p>\n<p>" + strings.Repeat("open the link to verify your address ", 4) + "</p>"

// part is one part of a received multipart message, its body decoded.
type part struct {
	contentType, encoding, body string
}

// parseAlternative checks that data is a multipart/alternative message and returns its parts in order.
func parseAlternative(t *testing.T, data string) []part {
	t.Helper()
	message, err := mail.ReadMessage(strings.NewReader(data))
	if err != nil {
		t.Fatalf("mail.ReadMessage = %v", err)
	}
	contentType := message.Header.Get("Content-Type")
	mediaType, parameters, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/alternative" || parameters["boundary"] == "" {
		t.Fatalf("Content-Type = %q (%v), want multipart/alternative with a boundary", contentType, err)
	}

	var parts []part
	reader := multipart.NewReader(message.Body, parameters["boundary"])
	for {
		raw, err := reader.NextRawPart()
		if errors.Is(err, io.EOF) {
			return parts
		}
		if err != nil {
			t.Fatalf("NextRawPart = %v", err)
		}
		body, err := io.ReadAll(quotedprintable.NewReader(raw))
		if err != nil {
			t.Fatalf("reading a part = %v", err)
		}
		parts = append(parts, part{
			contentType: raw.Header.Get("Content-Type"),
			encoding:    raw.Header.Get("Content-Transfer-Encoding"),
			body:        strings.ReplaceAll(string(body), "\r\n", "\n"),
		})
	}
}

func messageWith(change func(message *mailer.Message)) mailer.Message {
	message := mailer.Message{
		To:      "reader@example.com",
		Subject: "Verify your address, Zoë",
		Text:    "Hello Zoë,\n.\n.open the link\nthanks",
		Kind:    "identity.verification",
	}
	if change != nil {
		change(&message)
	}
	return message
}

func configFor(address string, change func(config *mailer.Config)) mailer.Config {
	config := mailer.Config{
		Addr:    address,
		From:    "blog@example.com",
		Timeout: 2 * time.Second,
	}
	if change != nil {
		change(&config)
	}
	return config
}

// Send delivers a text message from the configured sender to the recipient, authenticates with PLAIN when a
// username is set, and fails, with an error that never repeats the recipient, when the server refuses the
// recipient, when STARTTLS is required and not offered, when the server is unreachable or silent past the
// timeout, and when the recipient is not an address.
func TestSMTPSend(t *testing.T) {
	tCases := []struct {
		name    string
		options smtpOptions
		config  func(config *mailer.Config)
		message mailer.Message
		check   func(t *testing.T, server *smtpServer, err error)
	}{
		{
			name:    "the message reaches the server from the configured sender",
			config:  func(config *mailer.Config) { config.From = "Blog <blog@example.com>" },
			message: messageWith(nil),
			check: func(t *testing.T, server *smtpServer, err error) {
				if err != nil {
					t.Fatalf("Send = %v, want nil", err)
				}
				received := server.envelope(t)
				if received.from != "blog@example.com" || received.to != "reader@example.com" {
					t.Errorf("envelope = %s -> %s, want blog@example.com -> reader@example.com",
						received.from, received.to)
				}

				got := parseMessage(t, received.data)
				from, _ := got.header.AddressList("From")
				if len(from) != 1 || from[0].Name != "Blog" || from[0].Address != "blog@example.com" {
					t.Errorf("From = %v, want Blog <blog@example.com>", from)
				}
				to, _ := got.header.AddressList("To")
				if len(to) != 1 || to[0].Address != "reader@example.com" {
					t.Errorf("To = %v, want reader@example.com", to)
				}
				want := messageWith(nil)
				if subject, _ := new(mime.WordDecoder).DecodeHeader(got.header.Get("Subject")); subject != want.Subject {
					t.Errorf("Subject = %q, want %q", subject, want.Subject)
				}
				if contentType := got.header.Get("Content-Type"); contentType != "text/plain; charset=utf-8" {
					t.Errorf("Content-Type = %q, want text/plain; charset=utf-8", contentType)
				}
				if got.header.Get("Date") == "" || got.header.Get("Message-Id") == "" {
					t.Errorf("Date = %q, Message-Id = %q, want both set", got.header.Get("Date"), got.header.Get("Message-Id"))
				}
				if strings.TrimRight(got.body, "\n") != want.Text {
					t.Errorf("body = %q, want %q", got.body, want.Text)
				}
			},
		},
		{
			name:    "a message with HTML is sent as multipart/alternative with both parts",
			message: messageWith(func(message *mailer.Message) { message.HTML = htmlFixture }),
			check: func(t *testing.T, server *smtpServer, err error) {
				if err != nil {
					t.Fatalf("Send = %v, want nil", err)
				}

				got := parseAlternative(t, server.envelope(t).data)

				want := []part{
					{
						contentType: "text/plain; charset=utf-8",
						encoding:    "quoted-printable",
						body:        messageWith(nil).Text,
					},
					{
						contentType: "text/html; charset=utf-8",
						encoding:    "quoted-printable",
						body:        htmlFixture,
					},
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("parts = %+v, want %+v", got, want)
				}
			},
		},
		{
			name:    "a username makes the client authenticate with PLAIN",
			options: smtpOptions{advertiseAuth: true},
			config: func(config *mailer.Config) {
				config.Username, config.Password = "mailer", "s3cret"
			},
			message: messageWith(nil),
			check: func(t *testing.T, server *smtpServer, err error) {
				if err != nil {
					t.Fatalf("Send = %v, want nil", err)
				}
				if received := server.envelope(t); received.user != "mailer" || received.password != "s3cret" {
					t.Errorf("credentials = %q/%q, want mailer/s3cret", received.user, received.password)
				}
			},
		},
		{
			name:    "a server that refuses the recipient is an error",
			options: smtpOptions{refuseRcpt: true},
			message: messageWith(nil),
			check: func(t *testing.T, _ *smtpServer, err error) {
				if err == nil {
					t.Fatal("Send = nil, want an error")
				}
				if !strings.Contains(err.Error(), "550") {
					t.Errorf("Send = %q, want the reply code 550", err)
				}
				if strings.Contains(err.Error(), "reader@example.com") {
					t.Errorf("Send = %q, want no recipient in the error", err)
				}
			},
		},
		{
			name:    "STARTTLS that the server does not offer is an error",
			config:  func(config *mailer.Config) { config.StartTLS = true },
			message: messageWith(nil),
			check: func(t *testing.T, server *smtpServer, err error) {
				if err == nil || !strings.Contains(err.Error(), "STARTTLS") {
					t.Errorf("Send = %v, want an error naming STARTTLS", err)
				}
				select {
				case received := <-server.got:
					t.Errorf("the server received %+v, want nothing sent in the clear", received)
				default:
				}
			},
		},
		{
			name:    "a recipient that is not an address is an error",
			message: messageWith(func(message *mailer.Message) { message.To = "not an address" }),
			check: func(t *testing.T, _ *smtpServer, err error) {
				if err == nil {
					t.Fatal("Send = nil, want an error")
				}
				if strings.Contains(err.Error(), "not an address") {
					t.Errorf("Send = %q, want no recipient in the error", err)
				}
			},
		},
		{
			name:    "a subject cannot inject a header",
			message: messageWith(func(message *mailer.Message) { message.Subject = "Hi\r\nBcc: victim@example.com" }),
			check: func(t *testing.T, server *smtpServer, err error) {
				if err != nil {
					t.Fatalf("Send = %v, want nil", err)
				}
				if got := parseMessage(t, server.envelope(t).data); got.header.Get("Bcc") != "" {
					t.Errorf("Bcc = %q, want no injected header", got.header.Get("Bcc"))
				}
			},
		},
		{
			name:    "an unreachable server fails within the timeout",
			options: smtpOptions{closed: true},
			config:  func(config *mailer.Config) { config.Timeout = 200 * time.Millisecond },
			message: messageWith(nil),
			check: func(t *testing.T, _ *smtpServer, err error) {
				if err == nil {
					t.Error("Send = nil, want an error")
				}
			},
		},
		{
			name:    "a server that stops answering fails within the timeout",
			options: smtpOptions{stall: true},
			config:  func(config *mailer.Config) { config.Timeout = 200 * time.Millisecond },
			message: messageWith(nil),
			check: func(t *testing.T, _ *smtpServer, err error) {
				if err == nil {
					t.Error("Send = nil, want an error")
				}
			},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			server := startSMTPServer(t, tc.options)
			sender := mailer.NewSMTP(configFor(server.address, tc.config))

			start := time.Now()
			err := sender.Send(context.Background(), tc.message)

			if elapsed := time.Since(start); elapsed > time.Second {
				t.Errorf("Send took %v, want well under 1s", elapsed)
			}
			tc.check(t, server, err)
		})
	}
}

// Send stops when its context ends and returns that error.
func TestSMTPSendCancelled(t *testing.T) {
	server := startSMTPServer(t, smtpOptions{stall: true})
	sender := mailer.NewSMTP(configFor(server.address, func(config *mailer.Config) { config.Timeout = 10 * time.Second }))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := sender.Send(ctx, messageWith(nil))

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Send = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("Send took %v after its context ended, want well under 1s", elapsed)
	}
}
