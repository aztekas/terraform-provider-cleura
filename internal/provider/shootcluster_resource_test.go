package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccShootResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) }, //check username and token are defined
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				ConfigDirectory: config.TestStepDirectory(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify number of worker groups
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.#", "1"),
					// Verify Kubernetes version
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "kubernetes_version", "1.28.6"),
					// Verify first worker group in list
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.image_name", "gardenlinux"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.image_version", "1312.3.0"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.machine_type", "b.2c8gb"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.max_nodes", "2"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.min_nodes", "1"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.worker_group_name", "boboka"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.worker_node_volume_size", "50Gi"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("cleura_shoot_cluster.test", "uid"),
					resource.TestCheckResourceAttrSet("cleura_shoot_cluster.test", "last_updated"),
					resource.TestCheckResourceAttrSet("cleura_shoot_cluster.test", "hibernated"),
				),
			},
			// Update and Read testing
			{
				ConfigDirectory: config.TestStepDirectory(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify first order item updated
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.max_nodes", "3"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
