# Terraform

## Requirement

Run Terraform from this directory. `shell`-provider scripts depend on this pathing to work correctly.

## Cloud function code release / update logic

Including version into the zip-archive name seemingly fixes the problem (by activating both the deps update script & forcing new outputs from the `archive_file` resource, so there's no need for 2 separate `./tf apply` commands).

**TODO: check if excluding all the source code from the repository allows for correct Terraform commands running without getting an error.**

If source code is changed -> Terraform ignores the changes.

If `local.functions` function version is bumped (and source code is changed, otherwise it often does not really make any sense):
1. Terraform `null_resource` copies the updated source code into the build directory;
2. Terraform `archive_file` resource compresses the code and computes base64sha256 for the archive;
3. `yandex_function` resource is updated with the new `user_hash` and `version` tag.
