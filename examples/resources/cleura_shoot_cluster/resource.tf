resource "cleura_shoot_cluster" "test" {

  project            = "693jfssdjabn35495"
  region             = "sto2"
  name               = "my-cluster"
  kubernetes_version = "1.28.6"
  provider_details = {
    worker_groups = [
      {
        worker_group_name = "boboka" # max 6 characters
        machine_type      = "b.2c8gb"
        min_nodes         = 1
        max_nodes         = 2
      },
    ]
  }

  hibernation_schedules = [
    {
      start = "00 18 * * 1,2,3,4,5"
      end   = "00 08 * * 1,2,3,4,5"
    },
  ]
}
output "cluster" {
  value = cleura_shoot_cluster.test

}
