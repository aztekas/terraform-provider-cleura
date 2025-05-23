# STEP 1:
# Creates shoot cluster managing one worker group, with:
#   * Annotations
#   * Labels
#   * Taints
#   * Zones
# Set maintenance configuration with custom window times and machine image config to false

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
        max_nodes = 2
        annotations = {
          "tftestannotation": "abc",
          "test": "123"
        }
        labels = {
          "tftestlabel": "def"
        }
        taints = [{
          key    = "tftesttaint"
          value  = "456"
          effect = "NoExecute"
        }]
        zones = [
          "nova"
        ]
      },
    ]
  }
  maintenance = {
    auto_update_machine_image = false
    time_window_begin = "050000+0100"
    time_window_end = "060000+0100"
  }
}

// Set via CLEURA_TEST_PROJECT_ID environment variable in test suite
variable "project-id" {
}
