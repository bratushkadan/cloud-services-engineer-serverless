package ydynamo

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	"go.uber.org/zap"
)

const (
	TableEmailConfirmationTokens = "email_confirmation_tokens"
)

type EmailConfirmationRecord struct {
	Email     string    `dynamodbav:"email" json:"email"`
	Token     string    `dynamodbav:"token" json:"token"`
	ExpiresAt time.Time `dynamodbav:"expires_at" json:"expires_at"`
}

type EmailConfirmatorTokenRepo interface {
	InsertToken(ctx context.Context, email, token string) error
	ListTokensEmail(context context.Context, email string) ([]EmailConfirmationRecord, error)
	FindTokenRecord(context context.Context, token string) (*EmailConfirmationRecord, error)
}

var _ EmailConfirmatorTokenRepo = (*EmailConfirmator)(nil)

type EmailConfirmator struct {
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

func NewDynamoDbEmailConfirmator(ctx context.Context, accessKeyId, secretAccessKey string, ydbDocApiEndpoint string, logger *zap.Logger) (*EmailConfirmator, error) {
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

	return &EmailConfirmator{cl: client, l: logger}, nil
}

func (db *EmailConfirmator) InsertToken(ctx context.Context, email, token string) error {
	item := &dynamodb.PutItemInput{
		TableName: aws.String(TableEmailConfirmationTokens),
		Item: map[string]types.AttributeValue{
			"email":      &types.AttributeValueMemberS{Value: email},
			"token":      &types.AttributeValueMemberS{Value: token},
			"expires_at": &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Add(20*time.Minute).Unix(), 10)},
		},
	}

	output, err := db.cl.PutItem(ctx, item)
	if err != nil {
		return nil
	}

	var unmarshaledItem EmailConfirmationRecord
	if err := attributevalue.UnmarshalMap(output.Attributes, &unmarshaledItem); err != nil {
		return err
	}

	db.l.Info("inserted email token", zap.String("email", email))
	return nil
}

func (db *EmailConfirmator) ListTokensEmail(ctx context.Context, email string) ([]EmailConfirmationRecord, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(TableEmailConfirmationTokens),
		KeyConditionExpression: aws.String("email = :emailVal"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":emailVal": &types.AttributeValueMemberS{Value: email},
		},
	}

	result, err := db.cl.Query(ctx, input)
	if err != nil {
		return nil, err
	}

	var tokenRecords []EmailConfirmationRecord
	for _, item := range result.Items {
		var unmarshaledItem EmailConfirmationRecord
		if err := attributevalue.UnmarshalMap(item, &unmarshaledItem); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response from dynamodb: %v", err)
		}
		tokenRecords = append(tokenRecords, unmarshaledItem)
	}

	return tokenRecords, nil
}
func (db *EmailConfirmator) FindTokenRecord(ctx context.Context, token string) (*EmailConfirmationRecord, error) {
	filtEx := expression.Name("token").Equal(expression.Value(token))
	projEx := expression.NamesList(
		expression.Name("email"),
		expression.Name("token"),
		expression.Name("expires_at"),
	)

	expr, err := expression.NewBuilder().
		WithFilter(filtEx).
		WithProjection(projEx).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build find token record query: %v", err)
	}

	input := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		TableName:                 aws.String(TableEmailConfirmationTokens),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
	}
	result, err := db.cl.Scan(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	var unmarshaledItem EmailConfirmationRecord
	if err := attributevalue.UnmarshalMap(result.Items[0], &unmarshaledItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response from dynamodb: %v", err)
	}
	return &unmarshaledItem, nil
}
