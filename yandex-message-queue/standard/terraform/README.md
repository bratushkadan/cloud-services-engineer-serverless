# Terraform

## Requirement

Run Terraform from this directory. `shell`-provider scripts depend on this pathing to work correctly.

## Cloud function code release / update logic

Code changes cause source code zip archive file hashsum to change, thus Terraform apply triggers `yandex_function` resource's `user_hash` value update.

This is an expected (desired) behavior that indicates there's changes in source code and new cloud function version may not have the same version tag, i.e. version needs to be bumped before release.

> [!TIP]
> If you want to trigger function updates in Terraform only by dumping function `version` fields in `locals`, add `lifecycle.ignore_changed = [user_hash]` meta-attribute to the `yandex_function` Terraform resources.
