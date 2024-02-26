resource "cleura_shoot_cluster" "test" {
  project = "8ded727629b94bf7aaa2def479b54cfa"
  region = "sto2"
  name = "cleuratf-new"
  kubernetes_version = "1.28.6"
  provider_details = {
    worker_groups = [
	    {
        worker_group_name = "boboka"
        machine_type = "b.2c8gb"
        min_nodes = 1
        max_nodes = 3
        image_version = "1312.3.0"
      },
    ]
  }
}
