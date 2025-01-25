# Run function via CLI

## Prepare service account

### Create service account

```bash
export SERVICE_ACCOUNT_ID=$(yc iam service-account create \
  --name test-cloud-functions-sa \
  --description "Service account for testing Cloud Functions" \
  --format json |
  jq -cMr .id)
```

### Assign folder `editor` role to service account

```bash
export FOLDER_ID=$(yc config get folder-id)
yc resource-manager folder add-access-binding "${FOLDER_ID}" \
  --subject "serviceAccount:${SERVICE_ACCOUNT_ID}" \
  --role editor
```

## Prepare Cloud Function

### Create function

```bash
export CLOUD_FN_NAME="my-test-function"
yc serverless function create --name "${CLOUD_FN_NAME}"
```

### Create source code

```bash
export FN_PKG_SOURCE_CODE=$(mktemp /tmp/index.py)
cat <<EOF > "${FN_PKG_SOURCE_CODE}"
def handler(event, context):
    return {
        'statusCode': 200,
        'body': 'Hello World!',
    }
EOF
```

### Create function version

```bash
yc serverless function version create \
    --function-name "${CLOUD_FN_NAME}" \
    --memory 256m \
    --execution-timeout 5s \
    --runtime python312 \
    --entrypoint index.handler \
    --service-account-id $SERVICE_ACCOUNT_ID \
    --source-path "${FN_PKG_SOURCE_CODE}"
```

## Invoke Cloud Function

```bash
export FN_ID=$(yc serverless function list --format json \
  | jq -cMr ".[] | select(.name == \"${CLOUD_FN_NAME}\").id")
export FN_LATEST_VER_ID=$(yc serverless function version list \
  --function-name "${CLOUD_FN_NAME}" \
  --format json | jq -cMr ".[0].id")

yc serverless function invoke "${FN_ID}"
```

Output:
```json
{"statusCode": 200, "body": "Hello World!"}
```

## (Optionally) make function public

```bash
yc serverless function allow-unauthenticated-invoke "${CLOUD_FN_NAME}"
```
