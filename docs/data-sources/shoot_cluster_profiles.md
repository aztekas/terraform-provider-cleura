---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cleura_shoot_cluster_profiles Data Source - terraform-provider-cleura"
subcategory: ""
description: |-
  
---

# cleura_shoot_cluster_profiles (Data Source)



## Example Usage

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `filters` (Attributes) Filter output profile (see [below for nested schema](#nestedatt--filters))
- `gardener_domain` (String)

### Read-Only

- `gardenlinux_image_latest` (String)
- `kubernetes_latest` (String)
- `kubernetes_versions` (Attributes List) Available Kubernetes versions (see [below for nested schema](#nestedatt--kubernetes_versions))
- `machine_images` (Attributes List) Available machine images (see [below for nested schema](#nestedatt--machine_images))
- `machine_types` (Attributes List) Available machine types (see [below for nested schema](#nestedatt--machine_types))

<a id="nestedatt--filters"></a>
### Nested Schema for `filters`

Optional:

- `kubernetes` (Attributes) (see [below for nested schema](#nestedatt--filters--kubernetes))
- `machine_images` (Attributes) (see [below for nested schema](#nestedatt--filters--machine_images))
- `machine_types` (Attributes) (see [below for nested schema](#nestedatt--filters--machine_types))

<a id="nestedatt--filters--kubernetes"></a>
### Nested Schema for `filters.kubernetes`

Optional:

- `supported_only` (Boolean)


<a id="nestedatt--filters--machine_images"></a>
### Nested Schema for `filters.machine_images`

Optional:

- `supported_only` (Boolean)


<a id="nestedatt--filters--machine_types"></a>
### Nested Schema for `filters.machine_types`

Optional:

- `cpu` (String)
- `memory` (String)



<a id="nestedatt--kubernetes_versions"></a>
### Nested Schema for `kubernetes_versions`

Read-Only:

- `classification` (String)
- `expiration_date` (String)
- `version` (String)


<a id="nestedatt--machine_images"></a>
### Nested Schema for `machine_images`

Read-Only:

- `name` (String)
- `versions` (Attributes List) Version details (see [below for nested schema](#nestedatt--machine_images--versions))

<a id="nestedatt--machine_images--versions"></a>
### Nested Schema for `machine_images.versions`

Read-Only:

- `classification` (String)
- `expiration_date` (String)
- `version` (String)



<a id="nestedatt--machine_types"></a>
### Nested Schema for `machine_types`

Read-Only:

- `architecture` (String)
- `cpu` (String)
- `gpu` (String)
- `memory` (String)
- `name` (String)
- `usable` (Boolean)
