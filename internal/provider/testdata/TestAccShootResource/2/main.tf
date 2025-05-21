# STEP 2:
# Removes existing annotations, labels, taints and zone config from workergroup
# Adds a new worker group "newwg" to shoot cluster
# Modifies values of maintenance configuration block

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
      # add new worker group
      {
        worker_group_name = "newwg"
        machine_type = "b.2c4gb"
        min_nodes = 1
        max_nodes = 1
        annotations = {
          "annotatontest": "annotationtestvalue"
        }
        labels = {
          "labeltest": "labeltestvalue"
        }
        # taints = [{
        #   key = "tainttest"
        #   value = "tainttestvalue"
        #   effect = "NoSchedule"
        # }]
        zones = [
          "nova"
        ]
      },
    ]
  }
  hibernation_schedules = [
    {
      start = "00 18 * * 1,2,3,4,5"
      end   = "00 08 * * 1,2,3,4,5"
    },
  ]
  maintenance = {
    auto_update_kubernetes = false
    auto_update_machine_image = false
    time_window_begin = "020000+0100"
    time_window_end = "030000+0100"
  }
}

// Set via CLEURA_TEST_PROJECT_ID environment variable in test suite
variable "project-id" {
}
