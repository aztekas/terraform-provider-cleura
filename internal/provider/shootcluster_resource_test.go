package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccShootResource(t *testing.T) {
	projectID := os.Getenv("CLEURA_TEST_PROJECT_ID")
	varsTest := make(map[string]config.Variable)
	varsTest["project-id"] = config.StringVariable(projectID)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) }, // check username and token are defined
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				ConfigDirectory: config.TestStepDirectory(),
				ConfigVariables: varsTest,
				Check: resource.ComposeAggregateTestCheckFunc(

					// Verify number of worker groups
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.#", "1"),
					// Verify Kubernetes version
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "kubernetes_version", "1.32.4"),
					// Verify first worker group in list
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.image_name", "gardenlinux"),
					// resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.image_version", "1592.9.0"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.machine_type", "b.2c4gb"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.max_nodes", "2"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.min_nodes", "1"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.worker_group_name", "tstwg"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.worker_node_volume_size", "50Gi"),

					// Verify dynamic values have any value set in the state.
					// resource.TestCheckResourceAttrSet("cleura_shoot_cluster.test", "uid"), # 2025-05-18: Bug in API, does not return uid in response
					resource.TestCheckResourceAttrSet("cleura_shoot_cluster.test", "last_updated"),
					resource.TestCheckResourceAttrSet("cleura_shoot_cluster.test", "hibernated"),

					// Verify annotations, labels, taints and zones are set.
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.annotations.test", "123"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.labels.%", "1"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.labels.tftestlabel", "def"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.taints.#", "1"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.zones.#", "1"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.zones.0", "nova"),
				),
			},
			// Update and Read testing
			{
				ConfigDirectory: config.TestStepDirectory(),
				ConfigVariables: varsTest,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify max nodes has changed
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.max_nodes", "3"),

					// Verify fields has been removed
					resource.TestCheckNoResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.annotations.test"),
					resource.TestCheckNoResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.labels.tftestlabel"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
