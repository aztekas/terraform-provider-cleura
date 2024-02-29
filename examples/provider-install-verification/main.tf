terraform {
  required_providers {
    cleura = {
      source = "app.terraform.io/accelerate-at-iver/cleura"
    }
  }
}

provider "cleura" {}

data "cleura_shoot_cluster" "example" {
  name    = "cluster_name"
  project = "project_id"
  region  = "region"
}
