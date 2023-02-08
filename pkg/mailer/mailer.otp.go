package mailer

import (
	"fmt"
	"time"

	gomail "gopkg.in/mail.v2"
)

type OTPMailer interface {
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
	now := time.Now()
	configMessage := gomail.NewMessage()
	configMessage.SetHeader("From", mailer.email)
	configMessage.SetHeader("To", receiver)
	configMessage.SetHeader("Subject", "Verification Email")
	configMessage.SetBody("text/html",
		`<div style="font-family: Helvetica,Arial,sans-serif;min-width:1000px;overflow:auto;line-height:2">
			<div style="margin:50px auto;width:70%;padding:20px 0">
			<div style="border-bottom:1px solid #eee">
				<a href="" style="font-size:1.4em;color: #00466a;text-decoration:none;font-weight:600">Go Rest boilerplate</a>
			</div>
			<p style="font-size:1.1em">Hi,</p>
			<p>Thank you for choosing Our Services. Use the following OTP to complete your Sign Up procedures. OTP is valid for 5 minutes</p>
			<h2 style="background: #00466a;margin: 0 auto;width: max-content;padding: 0 10px;color: #fff;border-radius: 4px;">`+otpCode+`</h2>
			<p style="font-size:0.9em;">Regards,<br />Go Rest boilerplate</p>
			<hr style="border:none;border-top:1px solid #eee" />
			<div style="float:right;padding:8px 0;color:#aaa;font-size:0.8em;line-height:1;font-weight:300">
				<p>Copyright &copy; Go Rest boilerplate `+fmt.Sprintf("%d", now.Year())+`</p>
				<p>East Java, Indonesia</p>
			</div>
			</div>
		</div>
		`)

	dialer := gomail.NewDialer("smtp.gmail.com", 587, mailer.email, mailer.password)

	err = dialer.DialAndSend(configMessage)
	return
}
