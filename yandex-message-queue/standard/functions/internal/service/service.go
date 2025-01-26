package service

import (
	"context"
	"errors"
	"fmt"
	"fns/reg/internal/confirmer"
	"fns/reg/internal/ydynamo"
	"fns/reg/pkg/conf"
	"fns/reg/pkg/entity"
	"time"

	"go.uber.org/zap"
)

var (
	ErrInvalidConfirmationToken = errors.New("invalid confirmation token")
)

type EmailConfirmationAppConf struct {
	YdbDocApiEndpoint  string
	AwsAccessKeyId     string
	AwsSecretAccessKey string

	SenderEmail          string
	SenderPassword       string
	EmailConfirmationUrl string
}

func NewEmailConfirmationAppConf() *EmailConfirmationAppConf {
	return &EmailConfirmationAppConf{}
}

func (c *EmailConfirmationAppConf) LoadEnv() *EmailConfirmationAppConf {
	c.YdbDocApiEndpoint = conf.MustEnv("YDB_DOC_API_ENDPOINT")
	c.AwsAccessKeyId = conf.MustEnv("AWS_ACCESS_KEY_ID")
	c.AwsSecretAccessKey = conf.MustEnv("AWS_SECRET_ACCESS_KEY")

	c.SenderEmail = conf.MustEnv("SENDER_EMAIL")
	c.SenderPassword = conf.MustEnv("SENDER_PASSWORD")
	c.EmailConfirmationUrl = conf.MustEnv("EMAIL_CONFIRMATION_URL")
	return c
}

type EmailConfirmer interface {
	Confirm(ctx context.Context, token string) error
}

type EmailConfirmationSender interface {
	Send(ctx context.Context, email string) error
}

type EmailConfirmation struct {
	l         *zap.Logger
	repo      *ydynamo.DynamoDbEmailConfirmator
	confirmer *confirmer.EmailConfirmationLinkSender
}

func NewEmailConfirmation(conf *EmailConfirmationAppConf, logger *zap.Logger) (*EmailConfirmation, error) {
	confirmer := confirmer.NewEmailConfirmationLinkSender(conf.SenderEmail, conf.SenderPassword, conf.EmailConfirmationUrl)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	repo, err := ydynamo.NewDynamoDbEmailConfirmator(ctx, conf.AwsAccessKeyId, conf.AwsSecretAccessKey, conf.YdbDocApiEndpoint, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to setup dynamodb: %v", err)
	}

	return &EmailConfirmation{
		l:         logger,
		repo:      repo,
		confirmer: confirmer,
	}, nil
}

func (c *EmailConfirmation) Confirm(ctx context.Context, token string) error {
	c.l.Info("confirm email")
	c.l.Info("retrieve confirmation token records")
	record, err := c.repo.FindTokenRecord(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to retrieve tokens: %v", err)
	}
	if record == nil {
		c.l.Info("invalid confirmation token record")
		return ErrInvalidConfirmationToken
	}
	c.l.Info("retrieved confirmation token record", zap.String("email", record.Email))

	// FIXME: produce message here

	c.l.Info("confirmed email", zap.String("email", record.Email))
	return nil
}

func (c *EmailConfirmation) Send(ctx context.Context, email string) error {
	c.l.Info("create confirmation token and send email", zap.String("email", email))
	tokenString := entity.Id(64)
	err := c.repo.InsertToken(ctx, email, tokenString)
	if err != nil {
		return fmt.Errorf("failed to insert confirmation token: %v", err)
	}
	c.l.Info("inserted confirmation token", zap.String("email", email))

	if err := c.confirmer.Send(ctx, email, tokenString); err != nil {
		return fmt.Errorf("failed to send confirmation email: %v", err)
	}
	c.l.Info("sent confirmation email")

	return nil
}
