locals {
  project_root        = "${path.module}/.."
  fns_source_code_dir = "${local.project_root}/functions"
  build_dir           = "${local.project_root}/.zip"

  function_versions = {
    send_confirmation_email = "0.1.1"
    confirm_email           = "0.1.1"
    test_ydb                = "0.0.3"
  }

  functions = {
    send_confirmation_email = {
      version                = "v${replace(local.function_versions.send_confirmation_email, ".", "-")}"
      target_source_code_dir = "${local.build_dir}/send-confirmation-email"
      zip_path               = "${local.build_dir}/v${local.function_versions.send_confirmation_email}-send-confirmation-email.zip"
    }
    confirm_email = {
      version                = "v${replace(local.function_versions.confirm_email, ".", "-")}"
      target_source_code_dir = "${local.build_dir}/confirm-email"
      zip_path               = "${local.build_dir}/confirm-email-v${local.function_versions.confirm_email}.zip"
    }
    test_ydb = {
      version                = "v${replace(local.function_versions.test_ydb, ".", "-")}"
      target_source_code_dir = "${local.build_dir}/test-ydb"
      zip_path               = "${local.build_dir}/test-ydb-v${local.function_versions.test_ydb}.zip"
    }
  }
}


data "yandex_lockbox_secret" "app_sa_static_key" {
  secret_id = resource.yandex_lockbox_secret.app_sa_static_key.id
}
data "yandex_lockbox_secret" "email_provider" {
  name = "yandex-mail-provider"
}

locals {
  env = tomap({ for _, v in [
    "AWS_ACCESS_KEY_ID",
    "AWS_SECRET_ACCESS_KEY",
    "SENDER_EMAIL",
    "SENDER_PASSWORD",
    "EMAIL_CONFIRMATION_API_ENDPOINT",
    "YDB_DOC_API_ENDPOINT",
    "SQS_ENDPOINT",
  ] : v => v })

  lockbox_secrets = {
    send_confirmation_email = [
      {
        id                   = data.yandex_lockbox_secret.app_sa_static_key.id
        version_id           = data.yandex_lockbox_secret.app_sa_static_key.current_version[0].id
        key                  = "access_key_id"
        environment_variable = local.env.AWS_ACCESS_KEY_ID
      },
      {
        id                   = data.yandex_lockbox_secret.app_sa_static_key.id
        version_id           = data.yandex_lockbox_secret.app_sa_static_key.current_version[0].id
        key                  = "secret_access_key"
        environment_variable = local.env.AWS_SECRET_ACCESS_KEY
      },
      {
        id                   = data.yandex_lockbox_secret.email_provider.id
        version_id           = data.yandex_lockbox_secret.email_provider.current_version[0].id
        key                  = "email"
        environment_variable = local.env.SENDER_EMAIL
      },
      {
        id                   = data.yandex_lockbox_secret.email_provider.id
        version_id           = data.yandex_lockbox_secret.email_provider.current_version[0].id
        key                  = "password"
        environment_variable = local.env.SENDER_PASSWORD
      },
    ],
    confirm_email = [
      {
        id                   = data.yandex_lockbox_secret.app_sa_static_key.id
        version_id           = data.yandex_lockbox_secret.app_sa_static_key.current_version[0].id
        key                  = "access_key_id"
        environment_variable = local.env.AWS_ACCESS_KEY_ID
      },
      {
        id                   = data.yandex_lockbox_secret.app_sa_static_key.id
        version_id           = data.yandex_lockbox_secret.app_sa_static_key.current_version[0].id
        key                  = "secret_access_key"
        environment_variable = local.env.AWS_SECRET_ACCESS_KEY
      },
    ],
  }
}

# Recreate source code dir each time Terraform is run
data "shell_script" "user" {
  for_each = local.functions

  lifecycle_commands {
    read = file("${path.module}/scripts/shell-script-prepare-go-function.sh")
  }

  environment = {
    TARGET_SOURCE_CODE_DIR = each.value.target_source_code_dir
    SOURCE_CODE_DIR        = local.fns_source_code_dir
    FN_NAME                = each.key
    FN_VER                 = each.value.version
  }
}

resource "archive_file" "functions" {
  for_each = local.functions

  source_dir  = data.shell_script.user[each.key].output["source_dir"]
  output_path = data.shell_script.user[each.key].output["zip_path"]
  type        = "zip"
}

// TODO: delete manager SA
resource "yandex_iam_service_account" "cloud_functions_manager" {
  name        = "cloud-functions-manager"
  description = "service account for managing cloud functions"
}

resource "yandex_resourcemanager_folder_iam_member" "cloud_functions_manager_lockbox_payload_viewer" {
  folder_id = local.folder_id

  role   = "lockbox.payloadViewer"
  member = "serviceAccount:${yandex_iam_service_account.cloud_functions_manager.id}"
}

resource "yandex_function" "send_confirmation_email" {
  name        = "send-confirmation-email"
  description = "function for sending account email confirmation token via email"
  runtime     = "golang121"
  entrypoint  = "cmd/email-confirmation-sender-fn/handler.Handler"
  tags        = [local.functions.send_confirmation_email.version]
  user_hash   = archive_file.functions["send_confirmation_email"].output_base64sha256

  memory             = 128
  execution_timeout  = "10"
  service_account_id = yandex_iam_service_account.cloud_functions_manager.id

  environment = {
    (local.env.YDB_DOC_API_ENDPOINT)            = yandex_ydb_database_serverless.this.document_api_endpoint
    (local.env.EMAIL_CONFIRMATION_API_ENDPOINT) = local.auth_email_confirmation_api_endpoint
  }

  dynamic "secrets" {
    for_each = toset(local.lockbox_secrets.send_confirmation_email)
    content {
      id                   = secrets.value.id
      version_id           = secrets.value.version_id
      key                  = secrets.value.key
      environment_variable = secrets.value.environment_variable
    }
  }

  content {
    zip_filename = archive_file.functions["send_confirmation_email"].output_path
  }

  depends_on = [
    yandex_resourcemanager_folder_iam_member.app_lockbox_payload_viewer,
  ]

  lifecycle {
    ignore_changes = [user_hash]
  }
}

resource "yandex_function" "confirm_email" {
  name        = "confirm-email"
  description = "function for confirming account email via token"
  runtime     = "golang121"
  entrypoint  = "cmd/confirm-email-fn/handler.Handler"
  tags        = [local.functions.confirm_email.version]
  user_hash   = archive_file.functions["confirm_email"].output_base64sha256

  memory             = 128
  execution_timeout  = "10"
  service_account_id = yandex_iam_service_account.cloud_functions_manager.id

  environment = {
    (local.env.YDB_DOC_API_ENDPOINT) = yandex_ydb_database_serverless.this.document_api_endpoint
    (local.env.SQS_ENDPOINT)         = yandex_message_queue.email_confirmation.id
  }

  dynamic "secrets" {
    for_each = toset(local.lockbox_secrets.confirm_email)
    content {
      id                   = secrets.value.id
      version_id           = secrets.value.version_id
      key                  = secrets.value.key
      environment_variable = secrets.value.environment_variable
    }
  }

  content {
    zip_filename = archive_file.functions["confirm_email"].output_path
  }

  depends_on = [
    yandex_resourcemanager_folder_iam_member.app_lockbox_payload_viewer,
  ]

  lifecycle {
    ignore_changes = [user_hash]
  }
}

resource "yandex_function" "test_ydb" {
  name        = "test-db"
  description = "function for testing ydb"
  runtime     = "golang121"
  entrypoint  = "cmd/ydb-example-fn/main.Handler"
  tags        = [local.functions.test_ydb.version]
  user_hash   = archive_file.functions["test_ydb"].output_base64sha256

  memory             = 128
  execution_timeout  = "10"
  service_account_id = yandex_iam_service_account.cloud_functions_manager.id

  environment = {
    YDB_ENDPOINT = yandex_ydb_database_serverless.this.ydb_full_endpoint
  }

  content {
    zip_filename = archive_file.functions["test_ydb"].output_path
  }

  timeouts {
    create = "10m"
    update = "10m"
  }

  depends_on = [
    yandex_resourcemanager_folder_iam_member.app_lockbox_payload_viewer,
  ]

  lifecycle {
    ignore_changes = [user_hash]
  }
}

resource "yandex_function_iam_binding" "send_confirmation_email_caller" {
  function_id = yandex_function.send_confirmation_email.id
  role        = "serverless.functions.invoker"

  members = ["serviceAccount:${yandex_iam_service_account.auth_caller.id}"]
}
resource "yandex_function_iam_binding" "confirm_email_caller" {
  function_id = yandex_function.confirm_email.id
  role        = "serverless.functions.invoker"

  members = ["serviceAccount:${yandex_iam_service_account.auth_caller.id}"]
}
resource "yandex_function_iam_binding" "test_ydb_caller" {
  function_id = yandex_function.test_ydb.id
  role        = "serverless.functions.invoker"

  members = ["serviceAccount:${yandex_iam_service_account.auth_caller.id}"]
}
