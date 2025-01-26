# Document API

## HTTP API

First create Serverless YDB.

Export endpoint:
```bash
export ENDPOINT=...
```

Create the table:
```bash
curl \
  -H 'X-Amz-Target: DynamoDB_20120810.CreateTable' \
  -H "Authorization: Bearer $(yc iam create-token)" \
  -H "Content-Type: application/json" \
  -d '{"TableName": "docapitest/series","AttributeDefinitions":[{"AttributeName": "series_id", "AttributeType": "N"},{"AttributeName": "title", "AttributeType": "S"}],"KeySchema":[{"AttributeName": "series_id", "KeyType": "HASH"},{"AttributeName": "title", "KeyType": "RANGE"}]}' \
  $ENDPOINT
```

Insert the data
```bash
curl \
  -H 'X-Amz-Target: DynamoDB_20120810.PutItem' \
  -H "Authorization: Bearer $(yc iam create-token)" \
  -H "Content-Type: application/json" \
  -d '{"TableName": "docapitest/series", "Item": {"series_id": {"N": "1"}, "title": {"S": "IT Crowd"}, "series_info": {"S": "The IT Crowd is a British sitcom produced by Channel 4, written by Graham Linehan, produced by Ash Atalla and starring Chris ODowd, Richard Ayoade, Katherine Parkinson, and Matt Berry."}, "release_date": {"S": "2006-02-03"}}}' \
  $ENDPOINT
```

## AWS CLI

### Prepare SA

Create SA, grant it `ydb.editor` role and create static key.

```bash
export AWS_ACCESS_KEY_ID=$(jq -Mr '.access_key.key_id' access-key.json)
export AWS_SECRET_ACCESS_KEY=$(jq -Mr '.secret' access-key.json)
export AWS_DEFAULT_REGION=ru-central1
```

### Create table
```bash
aws dynamodb create-table \
  --table-name docapitest/series \
  --attribute-definitions \
  AttributeName=series_id,AttributeType=N \
  AttributeName=title,AttributeType=S \
  --key-schema \
  AttributeName=series_id,KeyType=HASH \
  AttributeName=title,KeyType=RANGE \
  --endpoint $ENDPOINT
```

### Insert data

```bash
aws dynamodb put-item \
  --table-name docapitest/series \
  --item '{"series_id": {"N": "1"}, "title": {"S": "IT Crowd"}, "series_info": {"S": "The IT Crowd is a British sitcom produced by Channel 4, written by Graham Linehan, produced by Ash Atalla and starring Chris ODowd, Richard Ayoade, Katherine Parkinson, and Matt Berry."}, "release_date": {"S": "2006-02-03"}}' \
  --endpoint $ENDPOINT
  aws dynamodb put-item \
  --table-name docapitest/series \
  --item '{"series_id": {"N": "2"}, "title": {"S": "Silicon Valley"}, "series_info": {"S": "Silicon Valley is an American comedy television series created by Mike Judge, John Altschuler and Dave Krinsky."}, "release_date": {"S": "2014-04-06"}}' \
  --endpoint $ENDPOINT
```

### Read data from table

```bash
aws dynamodb get-item --consistent-read \
  --table-name docapitest/series \
  --key '{"series_id": {"N": "1"}, "title": {"S": "IT Crowd"}}' \
  --endpoint $ENDPOINT
```

### Select data using key

```bash
aws dynamodb query \
  --table-name docapitest/series \
  --key-condition-expression "series_id = :name" \
  --expression-attribute-values '{":name":{"N":"2"}}' \
  --endpoint $ENDPOINT
```

### Delete table

```bash
aws dynamodb delete-table \
  --table-name docapitest/series \
  --endpoint $ENDPOINT
```
