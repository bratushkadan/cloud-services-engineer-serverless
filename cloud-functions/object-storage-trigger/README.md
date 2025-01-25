# Object Storage Trigger

## Setup SA and misc

### Add SA permissions

For this service account creation see [run-function-via-cli](../run-function-via-cli)

```bash
export SA_NAME=test-cloud-functions-sa
export SERVICE_ACCOUNT_ID=$(yc iam service-account get --name "${SA_NAME}" --format json | jq -cMr .id)
export FOLDER_ID=$(yc config get folder-id)
yc resource-manager folder add-access-binding "${FOLDER_ID}" \
    --role storage.editor \
    --subject "serviceAccount:$SERVICE_ACCOUNT_ID"
```

### Generate static key

```bash
yc iam access-key create --service-account-name "${SA_NAME}" --format json > access-key.json
```

```bash
export STATIC_KEY=$(yc iam access-key create --service-account-name "${SA_NAME}")
```

### Create object storage

```bash
export BUCKET_NAME=object-storage-trigger-bucket-noo19afbefb381
yc storage bucket create \
  --name "$BUCKET_NAME" \
  --max-size "$((128 * 2 ** 20))" \
  --public-read \
  --public-list
```

### Create new function version

```bash
yc serverless function version create \
  --function-name "${CLOUD_FN_NAME}" \
  --memory 256m \
  --execution-timeout 5s \
  --runtime python312 \
  --entrypoint index.handler \
  --service-account-id $SERVICE_ACCOUNT_ID \
  --source-path ./code
```


### Create new function version (2)

In the following example `function version create` uses the source (code) from the latest version of the function.

Source: https://yandex.cloud/ru/docs/storage/tools/boto

```bash
export FN_LATEST_VER_ID=$(yc serverless function version list \
  --function-name "${CLOUD_FN_NAME}" \
  --format json | jq -cMr ".[0].id")
export AWS_ACCESS_KEY_ID=$(jq -Mr '.access_key.key_id' access-key.json)
export AWS_SECRET_ACCESS_KEY=$(jq -Mr '.secret' access-key.json)
yc serverless function version create \
  --function-name "${CLOUD_FN_NAME}" \
  --memory 256m \
  --execution-timeout 5s \
  --runtime python312 \
  --entrypoint index.handler \
  --service-account-id $SERVICE_ACCOUNT_ID \
  --source-version-id "${FN_LATEST_VER_ID}" \
  --environment AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
  --environment AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
  --environment AWS_DEFAULT_REGION=ru-central1 \
  --environment BUCKET_NAME=$BUCKET_NAME
```

### Invoke function

```bash
yc serverless function invoke --name "${CLOUD_FN_NAME}"
```
