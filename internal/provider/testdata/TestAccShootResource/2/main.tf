resource "cleura_shoot_cluster" "test" {
  project = var.project-id
  region = "sto2"
  name = "cleuratf-new"
  provider_details = {
    worker_groups = [
	    {
        worker_group_name = "tstwg"
        machine_type = "b.2c4gb"
        min_nodes = 1
        max_nodes = 3
      },
    ]
    hibernation_schedules = [
      {
        start = "00 18 * * 1,2,3,4,5"
        end   = "00 08 * * 1,2,3,4,5"
      },
    ]
  }
}

// Set via CLEURA_TEST_PROJECT_ID environment variable in test suite
variable "project-id" {
}
