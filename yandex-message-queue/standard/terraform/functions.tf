locals {
  project_root        = "${path.module}/.."
  fns_source_code_dir = "${local.project_root}/functions"
  build_dir           = "${local.project_root}/.zip"

  fns = yamldecode(file("${path.module}/../config.yaml"))

  functions_versions_raw = {
    send_confirmation_email = "0.1.0"
    test_ydb                = "0.0.2-rc02.02.2025"
  }

  functions = {
    send_confirmation_email = {
      version                = "v${replace(local.functions_versions_raw.send_confirmation_email, ".", "-")}"
      target_source_code_dir = "${local.build_dir}/send-confirmation-email"
      zip_path               = "${local.build_dir}/v${local.functions_versions_raw.send_confirmation_email}-send-confirmation-email.zip"
    }
    test_ydb = {
      version                = "v${replace(local.functions_versions_raw.test_ydb, ".", "-")}"
      target_source_code_dir = "${local.build_dir}/test-ydb"
      zip_path               = "${local.build_dir}/test-ydb-v${local.functions_versions_raw.test_ydb}.zip"
    }
  }
}
output "fns" {
  value = local.fns
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
    "EMAIL_CONFIRMATION_URL",
    "YDB_DOC_API_ENDPOINT",
    "SQS_ENDPOINT",
  ] : v => v })

  lockbox_secrets = {
    confirmation_email = [
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
    ]
  }
}

resource "null_resource" "pack_code_send_confirmation_email" {
  provisioner "local-exec" {
    command = file("${path.module}/scripts/prepare-go-function.sh")

    environment = {
      TARGET_SOURCE_CODE_DIR = local.functions.send_confirmation_email.target_source_code_dir
      SOURCE_CODE_DIR        = local.fns_source_code_dir
    }
  }
  triggers = {
    run_on_version_update = local.functions.send_confirmation_email.version
    // always_run            = timestamp()
  }
}


resource "null_resource" "pack_function_code" {
  for_each = local.functions

  provisioner "local-exec" {
    command = file("${path.module}/scripts/prepare-go-function.sh")

    environment = {
      TARGET_SOURCE_CODE_DIR = each.value.target_source_code_dir
      SOURCE_CODE_DIR        = local.fns_source_code_dir
    }
  }
  triggers = {
    run_on_version_update = each.value.version
    // always_run            = timestamp()
  }
}

resource "archive_file" "functions" {
  for_each = local.functions

  source_dir  = each.value.target_source_code_dir
  output_path = each.value.zip_path
  type        = "zip"

  depends_on = [null_resource.pack_function_code]
}


resource "yandex_iam_service_account" "cloud_functions_manager" {
  name        = "cloud-functions-manager"
  description = "service account for managing cloud functions"
}
// resource "yandex_resourcemanager_folder_iam_member" "cloud_functions_manager_folder_editor" {
//   folder_id = local.folder_id
// 
//   role   = "editor"
//   member = "serviceAccount:${yandex_iam_service_account.cloud_functions_manager.id}"
// }
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
    (local.env.YDB_DOC_API_ENDPOINT)   = yandex_ydb_database_serverless.this.document_api_endpoint
    (local.env.EMAIL_CONFIRMATION_URL) = "foo.bar"
  }

  dynamic "secrets" {
    for_each = toset(local.lockbox_secrets.confirmation_email)
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
    // TODO: turn back on 
    // yandex_resourcemanager_folder_iam_member.cloud_functions_manager_folder_editor,
    yandex_resourcemanager_folder_iam_member.cloud_functions_manager_lockbox_payload_viewer,
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
    // TODO: turn back on 
    // yandex_resourcemanager_folder_iam_member.cloud_functions_manager_folder_editor,
    yandex_resourcemanager_folder_iam_member.cloud_functions_manager_lockbox_payload_viewer,
  ]
}

// There'll be an inevitable circular dependency "Cloud Function <--> API Gateway"
// due to the Cloud Function having need in API Gateway url in order to generate
// confirmation urls.
// resource "yandex_function" "confirm_email" {}
