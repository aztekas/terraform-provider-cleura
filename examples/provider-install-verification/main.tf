terraform {
  required_providers {
    cleura = {
      source = "aztek.no/aai/cleura"
    }
  }
}

provider "cleura" {}

data "cleura_shoot_cluster" "example" {
  name    = "cluster_name"
  project = "project_id"
  region  = "region"
}
