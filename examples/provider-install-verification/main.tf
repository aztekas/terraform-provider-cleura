terraform {
  required_providers {
    cleura = {
      source = "hashicorp.com/edu/cleura"
    }
  }
}

provider "cleura" {}

data "cleura_shoot_cluster" "example" {}
