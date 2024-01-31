terraform {
  required_providers {
    cleura = {
      source = "hashicorp.com/edu/cleura"
    }
  }
}

provider "cleura" {
  host     = "http://localhost:8088"
  username = "boo"
  password = "baa"
}

data "cleura_shoot_cluster" "edu" {
  project = "8ded727629b94bf7aaa2def479b54cfa"
  region = "sto2"
  name = "cleura-nais"
}

output "name" {
  value = data.cleura_shoot_cluster.edu
}
