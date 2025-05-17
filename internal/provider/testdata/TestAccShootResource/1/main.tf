
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
}

// Set via CLEURA_TEST_PROJECT_ID environment variable in test suite
variable "project-id" {
}
