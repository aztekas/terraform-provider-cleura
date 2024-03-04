# Gardener service must be enabled in the project
data "openstack_identity_project_v3" "gardener_project" {
  provider = openstack
  name     = "My project"
}

resource "cleura_shoot_cluster" "test" {
  project            = data.openstack_identity_project_v3.gardener_project.id
  region             = "sto2"
  name               = "my-cluster"
  kubernetes_version = "1.28.6"
  provider_details = {
    worker_groups = [
      {
        worker_group_name = "shdjkp" # max 6 characters
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
