resource "yandex_ydb_database_serverless" "this" {
  name        = "${local.common_name}-db"
  description = "serverless ydb for serverless type standard practicum course tests"
}

resource "yandex_iam_service_account" "app" {
  name        = "${local.common_name}-app"
  description = "serverless ymq type standard practicum course tests application sa"
}

resource "yandex_iam_service_account" "auth_caller" {
  name        = "${local.common_name}-auth-functions-caller"
  description = "serverless ymq type standard practicum course tests auth functions caller sa"
}

resource "yandex_resourcemanager_folder_iam_member" "app_lockbox_payload_viewer" {
  folder_id = local.folder_id

  role   = "lockbox.payloadViewer"
  member = "serviceAccount:${yandex_iam_service_account.app.id}"
}
resource "yandex_resourcemanager_folder_iam_member" "app_ydb_writer" {
  folder_id = local.folder_id

  role   = "ydb.editor"
  member = "serviceAccount:${yandex_iam_service_account.app.id}"
}
resource "yandex_resourcemanager_folder_iam_member" "app_ymq_reader" {
  folder_id = local.folder_id

  role   = "ymq.reader"
  member = "serviceAccount:${yandex_iam_service_account.app.id}"
}
resource "yandex_resourcemanager_folder_iam_member" "app_ymq_writer" {
  folder_id = local.folder_id

  role   = "ymq.writer"
  member = "serviceAccount:${yandex_iam_service_account.app.id}"
}
resource "yandex_resourcemanager_folder_iam_member" "app_serverless_mdb_user" {
  folder_id = local.folder_id

  role   = "serverless.mdbProxies.user"
  member = "serviceAccount:${yandex_iam_service_account.app.id}"
}

resource "yandex_lockbox_secret" "app_sa_static_key" {
  name        = "${local.common_name}-app-sa-static-key-secret"
  description = "static key secret for application sa for serverless ymq type standard practicum course lab"
}

resource "yandex_iam_service_account_static_access_key" "app_sa" {
  service_account_id = yandex_iam_service_account.app.id
  description        = "static access key for app sa"

  output_to_lockbox {
    secret_id            = yandex_lockbox_secret.app_sa_static_key.id
    entry_for_access_key = "access_key_id"
    entry_for_secret_key = "secret_access_key"
  }
}

resource "yandex_iam_service_account" "ydb_ymq_manager" {
  name        = "${local.common_name}-ydb-ymq-manager"
  description = "ymq and ydb manager for serverless ymq type standard practicum course tests"
}
resource "yandex_resourcemanager_folder_iam_member" "manager_ydb_admin" {
  folder_id = local.folder_id

  role   = "ydb.admin"
  member = "serviceAccount:${yandex_iam_service_account.ydb_ymq_manager.id}"
}
resource "yandex_resourcemanager_folder_iam_member" "manager_ymq_admin" {
  folder_id = local.folder_id

  role   = "ymq.admin"
  member = "serviceAccount:${yandex_iam_service_account.ydb_ymq_manager.id}"
}

resource "yandex_lockbox_secret" "ydb_ymq_manager_sa_static_key" {
  name        = "${local.common_name}-ydb-ymq-manager-sa-static-key"
  description = "static key secret for ydb/ymq manager sa for serverless ymq type standard practicum course lab"
}

resource "yandex_iam_service_account_static_access_key" "ydb_ymq_manager_sa" {
  service_account_id = yandex_iam_service_account.ydb_ymq_manager.id
  description        = "static access key for ydb/ymq management"

  output_to_lockbox {
    secret_id            = yandex_lockbox_secret.ydb_ymq_manager_sa_static_key.id
    entry_for_access_key = "access_key_id"
    entry_for_secret_key = "secret_access_key"
  }
}

resource "yandex_message_queue" "email_confirmation" {
  name = local.sqs_queues.email_confirmation

  visibility_timeout_seconds = 30

  redrive_policy = jsonencode({
    deadLetterTargetArn = yandex_message_queue.email_confirmation_dmq.arn
    maxReceiveCount     = 5
  })
}
resource "yandex_message_queue" "email_confirmation_dmq" {
  name = "${local.sqs_queues.email_confirmation}-dmq"
}
