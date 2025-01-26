package sender

import (
	"context"
	"fmt"
	"fns/reg/pkg/email"
)

type ConfirmationEmailBodyCreator struct {
	Url string
}

func (c ConfirmationEmailBodyCreator) Body(token string) string {
	return fmt.Sprintf("Follow the link to confirm the email address: %s?token=%s", c.Url, token)
}

type EmailConfirmationLinkSender struct {
	p  *email.YandexMailProvider
	bc ConfirmationEmailBodyCreator
}

func NewEmailConfirmationLinkSender(senderMail, senderPass, confirmationUrl string) *EmailConfirmationLinkSender {
	return &EmailConfirmationLinkSender{
		p:  email.NewYandexMailProvider(senderMail, senderPass),
		bc: ConfirmationEmailBodyCreator{Url: confirmationUrl},
	}
}

func (s EmailConfirmationLinkSender) Send(ctx context.Context, recipientEmail, token string) {
	s.p.SendMail(ctx, email.EmailContents{
		To:      recipientEmail,
		Subject: "Email confirmation",
		Body:    s.bc.Body(token),
	})
}
