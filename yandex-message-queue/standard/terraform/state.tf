terraform {
  backend "s3" {
    endpoint   = "storage.yandexcloud.net"
    bucket     = "serverless-ymq-type-standard-tf-state-zpds2x62re"
    key        = "serverless-ymq-type-standard-tf-state-zpds2x62re-state.tfstate"
    region     = "us-east-1"
    access_key = "YCAJEnenWDlDY5OVaozN4rzkH"

    skip_credentials_validation = true
    skip_metadata_api_check     = true
  }
}
