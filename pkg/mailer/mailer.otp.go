package mailer

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"time"

	gomail "gopkg.in/mail.v2"
)

// Embedded so the binary is self-contained — distroless runtime has
// no shell or filesystem to load templates from at deploy time.
//
//go:embed templates/*.html
var templatesFS embed.FS

// otpTpl is parsed once at package init. html/template (not text)
// auto-escapes the OTP code, defending against an attacker
// somehow injecting markup into the OTP path.
var otpTpl = template.Must(template.ParseFS(templatesFS, "templates/otp.html"))

// otpTemplateData feeds the template. AppName / Region are constants
// today; pulling them out makes white-labeling and i18n a config
// change rather than a code edit.
type otpTemplateData struct {
	AppName      string
	Region       string
	Code         string
	Year         int
	ValidMinutes int
}

const (
	defaultAppName      = "Go Rest boilerplate"
	defaultRegion       = "East Java, Indonesia"
	defaultValidMinutes = 5
)

type OTPMailer interface {
	// SendOTP delivers the OTP code to the receiver's inbox via the
	// configured SMTP relay. The async wrapper (AsyncOTPMailer) makes
	// this non-blocking from the request path; the inner sync impl
	// blocks for the SMTP round-trip.
	SendOTP(otpCode string, receiver string) (err error)
}

type otpMailer struct {
	email    string
	password string
}

func NewOTPMailer(email, password string) OTPMailer {
	return &otpMailer{
		email:    email,
		password: password,
	}
}

func (mailer *otpMailer) SendOTP(otpCode string, receiver string) (err error) {
	body, err := renderOTPBody(otpCode)
	if err != nil {
		return fmt.Errorf("render otp template: %w", err)
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", mailer.email)
	msg.SetHeader("To", receiver)
	msg.SetHeader("Subject", "Verification Email")
	msg.SetBody("text/html", body)

	dialer := gomail.NewDialer("smtp.gmail.com", 587, mailer.email, mailer.password)
	dialer.Timeout = 10 * time.Second

	return dialer.DialAndSend(msg)
}

// renderOTPBody is exported as a helper for tests so they can assert
// on the rendered HTML without spinning up an SMTP dialer.
func renderOTPBody(code string) (string, error) {
	var buf bytes.Buffer
	data := otpTemplateData{
		AppName:      defaultAppName,
		Region:       defaultRegion,
		Code:         code,
		Year:         time.Now().Year(),
		ValidMinutes: defaultValidMinutes,
	}
	if err := otpTpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
