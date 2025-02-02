terraform {
  required_providers {
    yandex = {
      source = "yandex-cloud/yandex"
    }
    shell = {
      source  = "scottwinkler/shell"
      version = ">=1.7.7"
    }
    external = {}
  }
}
