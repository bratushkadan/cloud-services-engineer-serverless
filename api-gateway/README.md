# API Gateway

## Prepare Cloud Function first

### Create

```bash
export SERVICE_ACCOUNT_ID=$(yc iam service-account create \
  --name "test-cloud-functions-sa-2" \
  --description "Service account for testing API Gateway with Cloud Functions" \
  --format json |
  jq -cMr .id)
```

### Assign role

```bash
export FOLDER_ID=$(yc config get folder-id)
yc resource-manager folder add-access-binding "${FOLDER_ID}" \
  --subject "serviceAccount:${SERVICE_ACCOUNT_ID}" \
  --role "functions.functionInvoker"
```

### Create function

```bash
export CLOUD_FN_NAME="api-gw-test-fn"
yc serverless function create --name "${CLOUD_FN_NAME}"
export FN_ID=$(yc serverless function get --name $CLOUD_FN_NAME | yq .id)
```

### Create function version

```bash
export FN_PKG_DIR=$(mktemp -d)
FN_CODE_DIR="$FN_PKG_DIR/cmd/python-handler"
mkdir -p "${FN_CODE_DIR}"
cat <<EOF > "${FN_CODE_DIR}/index.py"
import datetime
import json

def handler(event, context):
    current_time = datetime.datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S')

    message = 'Hello world'
    if event != None and 'user' in event["params"]:
        message = f'Hello, {event["params"]["user"]}'


    return {
        'statusCode': 200,
        'body': {
            'message': message,
            'time': current_time
        }
    }
EOF
yc serverless function version create \
    --function-name "${CLOUD_FN_NAME}" \
    --memory 128m \
    --execution-timeout 1s \
    --runtime python312 \
    --entrypoint "cmd/python-handler/index.handler" \
    --source-path "${FN_PKG_DIR}"
```

### Invoke function

```bash
yc serverless function invoke "${FN_ID}"
```

## Create

```bash
export API_GW_NAME="test-python-handler-api-gw"
export API_GW_SPEC_PATH="$(mktemp -d)/spec.yaml"
cat spec.yaml | envsubst > "${API_GW_SPEC_PATH}"
yc serverless api-gateway create \
  --name "${API_GW_NAME}" \
  --spec "${API_GW_SPEC_PATH}" \
  --description "API Gateway for testing Cloud Functions Python Handler"
```
