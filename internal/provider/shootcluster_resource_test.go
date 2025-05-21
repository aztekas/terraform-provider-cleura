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
					// resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "kubernetes_version", "1.32.4"),
					// Verify first worker group in list
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.image_name", "gardenlinux"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.machine_type", "b.2c4gb"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.max_nodes", "2"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.min_nodes", "1"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.worker_group_name", "tstwg"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.worker_node_volume_size", "50Gi"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("cleura_shoot_cluster.test", "uid"),
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

					// Verify maintenance configuration
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "maintenance.auto_update_kubernetes", "true"),     // Default value, not set in config
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "maintenance.auto_update_machine_image", "false"), // Manually set in main.tf
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "maintenance.time_window_begin", "050000+0100"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "maintenance.time_window_end", "060000+0100"),
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

					// Check the second worker group exists and properties are set
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.1.worker_group_name", "newwg"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.1.annotations.annotatontest", "annotationtestvalue"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.1.labels.labeltest", "labeltestvalue"),
					// resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.1.taints.#", "1"),
					// resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.1.taints.0.key", "taittest"),
					// resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.1.taints.0.value", "tainttestvalue"),
					// resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.1.taints.0.effect", "NoSchedule"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.1.zones.#", "1"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.1.zones.0", "nova"),

					// Verify updated maintenance config
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "maintenance.auto_update_kubernetes", "false"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "maintenance.auto_update_machine_image", "false"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "maintenance.time_window_begin", "020000+0100"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "maintenance.time_window_end", "030000+0100"),

					// Verify maintenance config was added
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "hibernation_schedules.#", "1"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "hibernation_schedules.0.start", "00 18 * * 1,2,3,4,5"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "hibernation_schedules.0.end", "00 08 * * 1,2,3,4,5"),
				),
			},
			{
				ConfigDirectory: config.TestStepDirectory(),
				ConfigVariables: varsTest,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.annotations.annotationsetfromupdate", "defg"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.labels.labelsetfromupdate", "hijk"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.taints.#", "1"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.taints.0.key", "taintsetfromupdate"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.taints.0.value", "789"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.taints.0.effect", "NoSchedule"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.zones.#", "1"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.0.zones.0", "nova"),

					// Verify only one worker group exists
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "provider_details.worker_groups.#", "1"),

					// Verify maintenance config has reverted to default
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "maintenance.auto_update_kubernetes", "true"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "maintenance.auto_update_machine_image", "true"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "maintenance.time_window_begin", "000000+0100"),
					resource.TestCheckResourceAttr("cleura_shoot_cluster.test", "maintenance.time_window_end", "010000+0100"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
