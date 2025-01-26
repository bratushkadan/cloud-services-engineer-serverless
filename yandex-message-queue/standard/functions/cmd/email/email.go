package main

import (
	"context"
	"fns/reg/internal/sender"
	"fns/reg/pkg/conf"
	"fns/reg/pkg/ydynamo"
	"log"

	"go.uber.org/zap"
)

var (
	ydbDocApiEndpoint  = conf.MustEnv("YDB_DOC_API_ENDPOINT")
	awsAccessKeyId     = conf.MustEnv("AWS_ACCESS_KEY_ID")
	awsSecretAccessKey = conf.MustEnv("AWS_SECRET_ACCESS_KEY")

	senderEmail          = conf.MustEnv("SENDER_EMAIL")
	senderPassword       = conf.MustEnv("SENDER_PASSWORD")
	emailConfirmationUrl = conf.MustEnv("EMAIL_CONFIRMATION_URL")
)

func main() {
	_ = sender.NewEmailConfirmationLinkSender(senderEmail, senderPassword, emailConfirmationUrl)
	ctx := context.Background()
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	db, err := ydynamo.New(ctx, awsAccessKeyId, awsSecretAccessKey, ydbDocApiEndpoint, logger)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.InsertEmailToken(ctx, "bratushkadan@gmail.com"); err != nil {
		log.Fatal(err)
	}

	if err := db.QueryDynamoDB(ctx, "bratushkadan@gmail.com"); err != nil {
		log.Fatal(err)
	}
}
