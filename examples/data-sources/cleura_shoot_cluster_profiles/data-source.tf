data "cleura_shoot_cluster_profiles" "profile" {
  filters = {
    machine_types = {
      cpu    = "2"
      memory = "4Gi"
    }
    machine_images = {
      supported_only = true
    }
    kubernetes = {
      supported_only = true
    }
  }
}

// Full profile data.
output "profiledata" {
  value = data.cleura_shoot_cluster_profiles.profile
}
// Machine types list, filtered respecting machine_types filter.
output "machine_types" {
  value = data.cleura_shoot_cluster_profiles.profile.machine_types

}
// Latest available (supported) kubernetes cluster version.
output "latest_kubernetes" {
  value = data.cleura_shoot_cluster_profiles.profile.kubernetes_latest
}
// Latest available (supported) garden linux image version.
output "latest_gardenlinux_image" {
  value = data.cleura_shoot_cluster_profiles.profile.gardenlinux_image_latest
}
