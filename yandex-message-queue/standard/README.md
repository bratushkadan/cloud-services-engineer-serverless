# Yandex Message Queue Lab – register, send confirmation email, validate confirmtion token & validate email

## YDB

Get serverless YDB

```bash
yc ydb database list
```

Get document API endpoint:
```bash
export YDB_DATABASE_ID=""
export YDB_DOC_API_ENDPOINT=$(yc ydb database get "${YDB_DATABASE_ID}" | yq .document_api_endpoint)
```

Create SA, grant it `ydb.editor` role and create static key.

```bash
export AWS_ACCESS_KEY_ID=$(jq -Mr '.access_key.key_id' access-key.json)
export AWS_SECRET_ACCESS_KEY=$(jq -Mr '.secret' access-key.json)
export AWS_DEFAULT_REGION=ru-central1
```

### Create `email_confirmation_tokens` database

In DynamoDB, you can define a composite primary key by specifying both a partition key (`HASH` key) and a sort key (`RANGE` key). This allows you to uniquely identify items in the table using a combination of these two attributes, which also facilitates more complex query patterns by allowing operations on sets of items with the same partition key.

```bash
export TABLE_CONF_TOKENS_NAME=email_confirmation_tokens
aws dynamodb create-table \
  --table-name "${TABLE_CONF_TOKENS_NAME}" \
  --attribute-definitions \
    AttributeName=token,AttributeType=S \
    AttributeName=email,AttributeType=S \
  --key-schema \
    AttributeName=email,KeyType=HASH \
    AttributeName=token,KeyType=RANGE \
  --endpoint "$YDB_DOC_API_ENDPOINT"
aws dynamodb update-time-to-live \
    --table-name "${TABLE_CONF_TOKENS_NAME}"  \
    --time-to-live-specification "Enabled=true, AttributeName=expires_at" \
  --endpoint "$YDB_DOC_API_ENDPOINT"
```


