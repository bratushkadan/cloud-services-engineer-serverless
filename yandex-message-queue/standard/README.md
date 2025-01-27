# Yandex Message Queue Lab – register, send confirmation email, validate confirmtion token & validate email

## Roadmap

- [ ] Setup YMQ
- [x] Implement Email Confirmation Sender
- [ ] Add YMQ publishing to the Email Confirmation Sender
- [ ] Implement Email Confirmer
- [ ] Implement Confirm User Account (possibly a mock service that just reads the value from the YMQ)
- [ ] Setup API Gateway
- [ ] Deploy & test the integrations

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

## Create YMQ

Create sa:
```bash
export YMQ_MANAGER_SA_NAME=ymq-manager
export YMQ_MANAGER_SA_ID=$(yc iam service-account create \
  --name "${YMQ_MANAGER_SA_NAME}" \
  --description "Yandex Message Queue managemet service account" \
  | yq -M .id
)
```

Grant role:
```bash
export FOLDER_ID=$(yc config get folder-id)
yc resource-manager folder add-access-binding \
  --id "${FOLDER_ID}" \
  --role "ymq.admin" \
  --subject "serviceAccount:${YMQ_MANAGER_SA_ID}"
```

Add access key for the management service account, create lockbox secret & version:
```bash
export YMQ_MANAGER_STATIC_KEY_LOCKBOX_SECRET_NAME="ymq-manager-static-key"
STATIC_KEY=$(yc iam access-key create --service-account-id "${YMQ_MANAGER_SA_ID}" --format json)
SECRET_PAYLOAD=$(bash -c "export AWS_ACCESS_KEY_ID=$(echo "${STATIC_KEY}" | jq -cMr .access_key.key_id); export AWS_SECRET_ACCESS_KEY=$(echo "${STATIC_KEY}" | jq -cMr .secret); cat ymq-lockbox-sa-static-key.tpl.yaml | envsubst")
yc lockbox secret create \
  --name "${YMQ_MANAGER_STATIC_KEY_LOCKBOX_SECRET_NAME}" \
  --description "Yandex Message Queue manager service account's static access key $(echo "${STATIC_KEY}" | jq .access_key.id)" \
  --payload "${SECRET_PAYLOAD}"
export YMQ_MANAGER_STATIC_KEY_LOCKBOX_SECRET_ID=$(yc lockbox secret get --name "${YMQ_MANAGER_STATIC_KEY_LOCKBOX_SECRET_NAME}" | yq -M .id)
```

Get secret payload for the management service account:
```bash
SECRET=$(yc lockbox payload get "${YMQ_MANAGER_STATIC_KEY_LOCKBOX_SECRET_ID}")
export AWS_ACCESS_KEY_ID=$(echo $SECRET | yq -M '.entries.[] | select(.key == "aws_access_key_id").text_value')
export AWS_SECRET_ACCESS_KEY=$(echo $SECRET | yq -M '.entries.[] | select(.key == "aws_secret_access_key").text_value')
```

Create queue via the management service account:
```bash
export SQS_QUEUE_NAME="email-confirmations"
aws sqs create-queue \
  --queue-name "${SQS_QUEUE_NAME}" \
  --endpoint "https://message-queue.api.cloud.yandex.net/"
```

### Create YMQ writer/reader service account
```bash
export YMQ_USER_SA_ID=aj***h
```

Grant it role for writing to queue:
```bash
yc resource-manager folder add-access-binding \
  --id "${FOLDER_ID}" \
  --role "ymq.reader" \
  --subject "serviceAccount:${YMQ_USER_SA_ID}"
yc resource-manager folder add-access-binding \
  --id "${FOLDER_ID}" \
  --role "ymq.writer" \
  --subject "serviceAccount:${YMQ_USER_SA_ID}"
```


