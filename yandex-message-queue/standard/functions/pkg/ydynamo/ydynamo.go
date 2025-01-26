package ydynamo

import (
	"context"
	"fmt"
	"fns/reg/pkg/entity"
	"net/url"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	"go.uber.org/zap"
)

type EmailConfirmationTokenRepo struct {
	cl *dynamodb.Client
	l  *zap.Logger
}

type ydbDocApiEndpointResolver struct {
	endpoint string
}

func (r ydbDocApiEndpointResolver) ResolveEndpoint(ctx context.Context, _ dynamodb.EndpointParameters) (smithyendpoints.Endpoint, error) {
	u, err := url.Parse(r.endpoint)
	if err != nil {
		return smithyendpoints.Endpoint{}, err
	}

	return smithyendpoints.Endpoint{
		URI: *u,
	}, nil
}

func newYdbDocApiEndpointResolver(endpoint string) *ydbDocApiEndpointResolver {
	return &ydbDocApiEndpointResolver{
		endpoint: endpoint,
	}
}

func New(ctx context.Context, accessKeyId, secretAccessKey string, ydbDocApiEndpoint string, logger *zap.Logger) (*EmailConfirmationTokenRepo, error) {
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("ru-central1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, secretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg, dynamodb.WithEndpointResolverV2(
		newYdbDocApiEndpointResolver(ydbDocApiEndpoint),
	))

	return &EmailConfirmationTokenRepo{cl: client, l: logger}, nil
}

func (db *EmailConfirmationTokenRepo) InsertEmailToken(ctx context.Context, email string) error {
	token := entity.Id(64)

	item := &dynamodb.PutItemInput{
		TableName: aws.String("email_confirmation_tokens"),
		Item: map[string]types.AttributeValue{
			"email":      &types.AttributeValueMemberS{Value: email},
			"token":      &types.AttributeValueMemberS{Value: token},
			"expires_at": &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Add(20*time.Minute).Unix(), 10)},
		},
	}

	_, err := db.cl.PutItem(ctx, item)
	if err != nil {
		return err
	}
	db.l.Info("inserted email token", zap.String("email", email))
	return nil
}

func (db *EmailConfirmationTokenRepo) QueryDynamoDB(ctx context.Context, email string) error {
	input := &dynamodb.QueryInput{
		TableName:              aws.String("email_confirmation_tokens"),
		KeyConditionExpression: aws.String("email = :emailVal"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":emailVal": &types.AttributeValueMemberS{Value: email},
		},
	}

	result, err := db.cl.Query(ctx, input)
	if err != nil {
		return err
	}

	for _, item := range result.Items {
		var unmarshaledItem struct {
			Email     string    `dynamodbav:"email" json:"email"`
			Token     string    `dynamodbav:"token" json:"token"`
			ExpiresAt time.Time `dynamodbav:"expires_at" json:"expires_at"`
		}
		if err := attributevalue.UnmarshalMap(item, &unmarshaledItem); err != nil {
			return fmt.Errorf("failed to unmarshal response from dynamodb: %v", err)
		}
		db.l.Info("query tokens unmarshaled item", zap.Any("token", unmarshaledItem))
	}

	return nil
}
