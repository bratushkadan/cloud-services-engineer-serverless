output "ydb" {
  value = {
    id                    = yandex_ydb_database_serverless.this.id
    document_api_endpoint = yandex_ydb_database_serverless.this.document_api_endpoint
  }
}

output "ymq" {
  value = {
    queues = {
      email_confirmation = {
        url = yandex_message_queue.email_confirmation.id
      }
    }
  }
}

output "ydb_ymq_manager_static_key_lockbox_secret_id" {
  value = yandex_lockbox_secret.ydb_ymq_manager_sa_static_key.id
}

output "app_sa" {
  value = {
    id                           = yandex_iam_service_account.app.id
    static_key_lockbox_secret_id = yandex_lockbox_secret.app_sa_static_key.id
  }
}
