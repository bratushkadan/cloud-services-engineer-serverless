package confirmer

import (
	"context"
	"fmt"
	"fns/reg/pkg/email"
)

type confirmationEmailBodyCreator struct {
	Url string
}

func (c confirmationEmailBodyCreator) Body(token string) string {
	return fmt.Sprintf("Follow the link to confirm the email address: %s?token=%s", c.Url, token)
}

type EmailConfirmationLinkSender struct {
	p  *email.YandexMailProvider
	bc confirmationEmailBodyCreator
}

func NewEmailConfirmationLinkSender(senderMail, senderPass, confirmationUrl string) *EmailConfirmationLinkSender {
	return &EmailConfirmationLinkSender{
		p:  email.NewYandexMailProvider(senderMail, senderPass),
		bc: confirmationEmailBodyCreator{Url: confirmationUrl},
	}
}

func (s EmailConfirmationLinkSender) Send(ctx context.Context, recipientEmail, token string) error {
	return s.p.SendMail(ctx, email.EmailContents{
		To:      recipientEmail,
		Subject: "Email confirmation",
		Body:    s.bc.Body(token),
	})
}
