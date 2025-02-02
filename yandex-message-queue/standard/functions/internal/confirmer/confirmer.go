package confirmer

import (
	"context"
	"fmt"
	"fns/reg/pkg/email"
	"net/url"
)

type ConfirmationUrlResolver = func(ctx context.Context) (*url.URL, error)

type confirmationEmailBodyCreator struct {
	resolver ConfirmationUrlResolver
}

func (c confirmationEmailBodyCreator) Body(ctx context.Context, token string) (string, error) {
	url, err := c.resolver(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to resolve email confirmation url for email body creator: %v", err)
	}

	q := url.Query()
	q.Add("token", token)
	url.RawQuery = q.Encode()

	return fmt.Sprintf("Follow the link to confirm the email address: %s", url.String()), nil
}

type Email struct {
	p  *email.YandexMailProvider
	bc confirmationEmailBodyCreator
}

func NewEmail(senderMail, senderPass string, resolver ConfirmationUrlResolver) *Email {
	return &Email{
		p:  email.NewYandexMailProvider(senderMail, senderPass),
		bc: confirmationEmailBodyCreator{resolver: resolver},
	}
}

func (s Email) Send(ctx context.Context, recipientEmail, token string) error {
	messageBody, err := s.bc.Body(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to create message body: %v", err)
	}

	return s.p.SendMail(ctx, email.EmailContents{
		To:      recipientEmail,
		Subject: "Email confirmation",
		Body:    messageBody,
	})
}
