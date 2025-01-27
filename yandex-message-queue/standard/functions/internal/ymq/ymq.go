package emconfmq

import (
	"context"
	"encoding/json"
	"fmt"
	"fns/reg/pkg/ymq"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.uber.org/zap"
)

type EmailConfirmationMq struct {
	ymq *ymq.Ymq
	l   *zap.Logger

	qUrl string
}

func New(emailConfirmationQueueUrl string, logger *zap.Logger) *EmailConfirmationMq {
	return &EmailConfirmationMq{l: logger, qUrl: emailConfirmationQueueUrl}
}

type EmailConfirmationDTO struct {
	Email string `json:"string"`
}

func (q *EmailConfirmationMq) GetConfirmations(ctx context.Context) (dto []EmailConfirmationDTO, deleteMessages func(ctx context.Context) error, err error) {
	output, err := q.ymq.Cl.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(q.qUrl),
		MaxNumberOfMessages: 50,
		WaitTimeSeconds:     5,
	})

	emailConfirmations := make([]EmailConfirmationDTO, 0, len(output.Messages))
	deleteMessageBatchReqEntries := make([]types.DeleteMessageBatchRequestEntry, 0, len(output.Messages))
	for _, message := range output.Messages {
		var emailConfirmation EmailConfirmationDTO
		if err := json.Unmarshal([]byte(*message.Body), &emailConfirmation); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal sqs email confirmation message: %v", err)
		}
		emailConfirmations = append(emailConfirmations, emailConfirmation)
		deleteMessageBatchReqEntries = append(deleteMessageBatchReqEntries, types.DeleteMessageBatchRequestEntry{
			Id:            message.MessageId,
			ReceiptHandle: message.ReceiptHandle,
		})
	}

	return emailConfirmations, func(ctx context.Context) error {
		_, err := q.ymq.Cl.DeleteMessageBatch(ctx, &sqs.DeleteMessageBatchInput{
			QueueUrl: aws.String(q.qUrl),
			Entries:  deleteMessageBatchReqEntries,
		})
		return err
	}, nil
}
func (q *EmailConfirmationMq) PublishConfirmation(ctx context.Context, conf EmailConfirmationDTO) error {
	return nil, nil
}
