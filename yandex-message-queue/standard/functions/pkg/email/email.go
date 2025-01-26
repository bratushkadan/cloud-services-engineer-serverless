package email

import (
	"context"

	"gopkg.in/gomail.v2"
)

type EmailPasswordProvider struct {
	d          *gomail.Dialer
	senderMail string
}

type EmailContents struct {
	To      string
	Subject string
	Body    string
}

func (p *EmailPasswordProvider) SendMail(ctx context.Context, email EmailContents) error {
	m := gomail.NewMessage()
	m.SetHeader("From", p.senderMail)
	m.SetHeader("To", email.To)
	m.SetHeader("Subject", email.Subject)

	m.SetBody("text/plain", email.Body)

	return p.d.DialAndSend(m)
}

type GmailProvider struct {
	p *EmailPasswordProvider
}
type YandexMailProvider struct {
	p *EmailPasswordProvider
}

func NewGmailProvider(senderMail, senderPass string) *GmailProvider {
	return &GmailProvider{
		p: &EmailPasswordProvider{
			d:          gomail.NewDialer("smtp.gmail.com", 587, senderMail, senderPass),
			senderMail: senderMail,
		},
	}
}
func NewYandexMailProvider(senderMail, senderPass string) *YandexMailProvider {
	return &YandexMailProvider{
		p: &EmailPasswordProvider{
			d:          gomail.NewDialer("smtp.yandex.com", 465, senderMail, senderPass),
			senderMail: senderMail,
		},
	}
}

func (p *GmailProvider) SendMail(ctx context.Context, email EmailContents) error {
	return p.p.SendMail(ctx, email)
}
func (p *YandexMailProvider) SendMail(ctx context.Context, email EmailContents) error {
	return p.p.SendMail(ctx, email)
}
