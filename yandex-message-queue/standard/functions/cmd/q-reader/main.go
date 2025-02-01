package main

import (
	"context"
	"errors"
	"fmt"
	"fns/reg/internal/emconfmq"
	"fns/reg/pkg/conf"
	"fns/reg/pkg/ymq"
	"log"

	"go.uber.org/zap"
)

func main() {
	var (
		SqsEndpoint        = conf.MustEnv("SQS_ENDPOINT")
		AwsAccessKeyId     = conf.MustEnv("AWS_ACCESS_KEY_ID")
		AwsSecretAccessKey = conf.MustEnv("AWS_SECRET_ACCESS_KEY")
	)

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	ymq, err := ymq.New(context.Background(), AwsAccessKeyId, AwsSecretAccessKey, SqsEndpoint, logger)
	if err != nil {
		logger.Error("failed to setup ymq", zap.Error(err))
	}
	mq := emconfmq.New(ymq, logger)

	if err := mq.ProcessConfirmations(context.Background(), func(ctx context.Context, dto []emconfmq.EmailConfirmationDTO) error {
		for _, v := range dto {
			fmt.Printf("%+v\n", v)
		}
		return errors.New("could not process message (fake error)")
	}); err != nil {
		logger.Fatal("error processing messages", zap.Error(err))
	}
}
