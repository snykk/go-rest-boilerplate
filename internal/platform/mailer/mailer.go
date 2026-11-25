// Package mailer sends plain-text email, with an optional HTML version, over SMTP. A Sender delivers a message
// and reports the outcome; a Queue sends in the background behind the Enqueuer interface, so a request never
// waits for the mail server; a Recorder keeps messages in memory for tests and for runs with no mail server.
package mailer

import "context"

// Message is one email: plain text, with an optional HTML version. The sender is not part of it: it is the
// mailer's configuration.
type Message struct {
	// To is the recipient's address, such as "reader@example.com".
	To string
	// Subject is the subject line. It may hold any text; characters outside ASCII are encoded.
	Subject string
	// Text is the body, in UTF-8.
	Text string
	// HTML is an HTML version of Text, in UTF-8, or empty for a message of text alone. A message with both is
	// sent as multipart/alternative, and the reader's mail client shows the version it prefers. Like the
	// recipient, the subject and Text, it is never logged.
	HTML string
	// Kind names the message in log records, such as "identity.verification". It is the only field of a
	// message that is ever logged.
	Kind string
}

// Sender delivers a message and waits for the outcome.
type Sender interface {
	// Send delivers message, or returns an error when it could not, including when ctx ends first. The error
	// never repeats the recipient, the subject or the body. It is safe for concurrent use.
	Send(ctx context.Context, message Message) error
}

// Enqueuer accepts a message for delivery later, without waiting for it.
type Enqueuer interface {
	// Enqueue takes message for delivery and returns at once. It reports nothing: a message that cannot be
	// delivered is logged, not returned. It is safe for concurrent use.
	Enqueue(ctx context.Context, message Message)
}
