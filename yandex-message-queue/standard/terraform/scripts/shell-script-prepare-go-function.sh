#!/bin/sh

set -e

function log_error() {
    echo "\n\n@@@ ERROR:$@" >&2
}

function validate_env_var() {
    local var_name="$1"

    if [ -z "${!var_name}" ]; then
        log_error "env \"$var_name\" is required"
        exit 1
    fi
}

validate_env_var TARGET_SOURCE_CODE_DIR
validate_env_var SOURCE_CODE_DIR
validate_env_var FN_NAME
validate_env_var FN_VER

# Terraform, and subsequently, this script is assumed to be run from the ./terraform directory

mkdir -p "$TARGET_SOURCE_CODE_DIR"

cp -r "$SOURCE_CODE_DIR"/* "$TARGET_SOURCE_CODE_DIR"
# delete go version line, as required by yc builder https://yandex.cloud/ru/docs/functions/lang/golang/dependencies#mod
sed -i'.bak' -E 's/^go\ [0-9]+\.[0-9]+\.[0-9]+$//' "$TARGET_SOURCE_CODE_DIR/go.mod"
rm "$TARGET_SOURCE_CODE_DIR/go.mod.bak"

# echo "{\"user\": \"$(whoami) $BOOGA\", \"source_dir\": \"$TARGET_SOURCE_CODE_DIR\", \"\":\"\"}"
cat <<EOF
{
  "user": "$(whoami) ${BOOGA}",
  "source_dir": "${TARGET_SOURCE_CODE_DIR}",
  "zip_path": "${TARGET_SOURCE_CODE_DIR}/${FN_NAME}-${FN_VER}.zip"
}
EOF
