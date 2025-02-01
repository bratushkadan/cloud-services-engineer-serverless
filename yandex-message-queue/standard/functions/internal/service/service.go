package service

import (
	"context"
	"errors"
	"fmt"
	"fns/reg/internal/confirmer"
	"fns/reg/internal/emconfmq"
	"fns/reg/internal/ydynamo"
	"fns/reg/pkg/conf"
	"fns/reg/pkg/entity"
	"fns/reg/pkg/ymq"
	"time"

	"go.uber.org/zap"
)

var (
	ErrInvalidConfirmationToken = errors.New("invalid confirmation token")
	ErrConfirmationTokenExpired = errors.New("confirmation token expired")
)

type EmailConfirmationAppConf struct {
	YdbDocApiEndpoint  string
	SqsEndpoint        string
	withSqs            bool
	AwsAccessKeyId     string
	AwsSecretAccessKey string

	SenderEmail          string
	SenderPassword       string
	EmailConfirmationUrl string

	emailConfirmationSendTimeout time.Duration
}

func NewEmailConfirmationAppConf() *EmailConfirmationAppConf {
	return &EmailConfirmationAppConf{}
}

func (c *EmailConfirmationAppConf) WithSqs() *EmailConfirmationAppConf {
	c.withSqs = true
	return c
}

func (c *EmailConfirmationAppConf) LoadEnv() *EmailConfirmationAppConf {
	c.YdbDocApiEndpoint = conf.MustEnv("YDB_DOC_API_ENDPOINT")
	if c.withSqs {
		c.SqsEndpoint = conf.MustEnv("SQS_ENDPOINT")
	}
	c.AwsAccessKeyId = conf.MustEnv("AWS_ACCESS_KEY_ID")
	c.AwsSecretAccessKey = conf.MustEnv("AWS_SECRET_ACCESS_KEY")

	c.SenderEmail = conf.MustEnv("SENDER_EMAIL")
	c.SenderPassword = conf.MustEnv("SENDER_PASSWORD")
	c.EmailConfirmationUrl = conf.MustEnv("EMAIL_CONFIRMATION_URL")

	c.emailConfirmationSendTimeout = 5 * time.Second

	return c
}

type EmailConfirmer interface {
	Confirm(ctx context.Context, token string) error
	Send(ctx context.Context, email string) error
}

type EmailConfirmation struct {
	conf      EmailConfirmationAppConf
	l         *zap.Logger
	repo      *ydynamo.DynamoDbEmailConfirmator
	sqs       *emconfmq.EmailConfirmationMq
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

	var sqs *emconfmq.EmailConfirmationMq
	if conf.withSqs {
		ymq, err := ymq.New(ctx, conf.AwsAccessKeyId, conf.AwsSecretAccessKey, conf.SqsEndpoint, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to setup ymq for email confirmation message queue: %v", err)
		}
		sqs = emconfmq.New(ymq, logger)
	}

	return &EmailConfirmation{
		l:         logger,
		repo:      repo,
		confirmer: confirmer,
		sqs:       sqs,
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
	if time.Now().After(record.ExpiresAt) {
		return ErrConfirmationTokenExpired
	}
	c.l.Info("validated confirmation token record", zap.String("email", record.Email))

	if err := c.sqs.PublishConfirmation(ctx, emconfmq.EmailConfirmationDTO{Email: record.Email}); err != nil {
		return fmt.Errorf("failed to produce confirmation message: %v", err)
	}
	c.l.Info("produced confirmation message", zap.String("email", record.Email))

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

	ctx, cancel := context.WithTimeout(ctx, c.conf.emailConfirmationSendTimeout)
	defer cancel()
	if err := c.confirmer.Send(ctx, email, tokenString); err != nil {
		return fmt.Errorf("failed to send confirmation email: %v", err)
	}
	c.l.Info("sent confirmation email")

	return nil
}
