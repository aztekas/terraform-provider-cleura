data "cleura_shoot_cluster" "test" {
  project = "<project-id>"
  region  = "sto2"
  name    = "cluster-name"
}

output "name" {
  value = data.cleura_shoot_cluster.test
}
