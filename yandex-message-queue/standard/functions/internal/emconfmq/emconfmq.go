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
}

func New(ymq *ymq.Ymq, logger *zap.Logger) *EmailConfirmationMq {
	return &EmailConfirmationMq{ymq: ymq, l: logger}
}

type EmailConfirmationDTO struct {
	Email string `json:"string"`
}

func (q *EmailConfirmationMq) ProcessConfirmations(ctx context.Context, process func(ctx context.Context, dto []EmailConfirmationDTO) error) error {
	output, err := q.ymq.Cl.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(q.ymq.Endpoint()),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     5,
	})
	if err != nil {
		return fmt.Errorf("failed to receive email confirmation messages: %v", err)
	}

	emailConfirmations := make([]EmailConfirmationDTO, 0, len(output.Messages))
	deleteMessageBatchReqEntries := make([]types.DeleteMessageBatchRequestEntry, 0, len(output.Messages))
	for _, message := range output.Messages {
		var emailConfirmation EmailConfirmationDTO
		if err := json.Unmarshal([]byte(*message.Body), &emailConfirmation); err != nil {
			return fmt.Errorf("failed to unmarshal sqs email confirmation message: %v", err)
		}
		emailConfirmations = append(emailConfirmations, emailConfirmation)
		deleteMessageBatchReqEntries = append(deleteMessageBatchReqEntries, types.DeleteMessageBatchRequestEntry{
			Id:            message.MessageId,
			ReceiptHandle: message.ReceiptHandle,
		})
	}

	err = process(ctx, emailConfirmations)
	// TODO: implement DMQ
	if err != nil {
		return fmt.Errorf("failed to process confirmations: %v", err)
	}

	_, err = q.ymq.Cl.DeleteMessageBatch(ctx, &sqs.DeleteMessageBatchInput{
		QueueUrl: aws.String(q.ymq.Endpoint()),
		Entries:  deleteMessageBatchReqEntries,
	})
	return fmt.Errorf("failed to delete processed messages from the message queue: %v", err)
}
func (q *EmailConfirmationMq) PublishConfirmation(ctx context.Context, conf EmailConfirmationDTO) error {
	emailConfirmationMsg, err := json.Marshal(&conf)
	if err != nil {
		return fmt.Errorf("failed to deserialize email confirmation dto: %v", err)
	}
	_, err = q.ymq.Cl.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(string(emailConfirmationMsg)),
		QueueUrl:    aws.String(q.ymq.Endpoint()),
	})
	return err
}
