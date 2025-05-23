# STEP 2:
# Adds annotations, labels, taints and zones back to workergroup (using update function)
# Removes worker group "newwg" from shoot cluster
# Removes maintenance configuration

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
        annotations = {
          "annotationsetfromupdate": "defg",
        }
        labels = {
          "labelsetfromupdate": "hijk"
        }
        taints = [{
          key    = "taintsetfromupdate"
          value  = "789"
          effect = "NoSchedule"
        }]
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
}

// Set via CLEURA_TEST_PROJECT_ID environment variable in test suite
variable "project-id" {
}
