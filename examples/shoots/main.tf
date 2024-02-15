
# data "cleura_shoot_cluster" "edu" {
#   project = "8ded727629b94bf7aaa2def479b54cfa"
#   region = "sto2"
#   name = "cleura-nais"
# }

# output "name" {
#   value = data.cleura_shoot_cluster.edu
# }


resource "cleura_shoot_cluster" "test" {

  project = "8ded727629b94bf7aaa2def479b54cfa"
  region = "sto2"
  name = "cleuratf-new"
  kubernetes_version = "1.28.6"
  provider_details = {
    //floating_pool_name = "ext-dev" # optional with default value
    #workers_cidr = "CIDR" # optional with default value
    worker_groups = [
      {
        #worker_group_name = "mywg" # optional
        machine_type = "b.2c8gb"# # required #
        //image_name = "" # optional with default value
        //image_version = "" # optional with default value
        //worker_node_volume_size = "" # optional with default value
        min_nodes = 1 # optional with default from cluera api
        max_nodes = 3 # optional with default from cleura api
      },
    ]
  }

  hibernation_schedules = [
	{
		start = "00 19 * * 1,2,3,4,5"
		end = "00 08 * * 1,2,3,4,5"
	},
	# {
	# 	start = "00 11 * * 1,2,3,4,5"
	# 	end = "00 13 * * 1,2,3,4,5"
	# },

  ]
}
#dd
output "cluster" {
	value = cleura_shoot_cluster.test

}

/*
shootCluster := cleuraclient.ShootClusterRequest{
		Shoot: cleuraclient.ShootClusterRequestConfig{
			Name: name,
			KubernetesVersion: cleuraclient.K8sVersion{
				Version: "1.28.6",
			},
			Provider: cleuraclient.ProviderDetails{
				InfrastructureConfig: cleuraclient.InfrastructureConfigDetails{
					FloatingPoolName: "ext-net",
				},
				Workers: []cleuraclient.Worker{
					{
						Machine: cleuraclient.MachineDetails{
							Type: "b.2c8gb",
							Image: cleuraclient.ImageDetails{
								Name:    "gardenlinux",
								Version: "1312.2.0",
							},
						},
						Volume: cleuraclient.VolumeDetails{
							Size: "50Gi",
						},
					},
				},
			},
			// Hibernation: &cleuraclient.HibernationSchedules{
			// 	HibernationSchedules: []cleuraclient.HibernationSchedule{
			// 		{
			// 			Start: "4324324",
			// 			End:   "423432",
			// 		},
			// 	},
			// },
		},
	}
*/
