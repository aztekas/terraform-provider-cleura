# Terraform Provider Cleura

Unofficial terraform provider for [*Cleura The European Cloud*](https://cleura.com/) public cloud. Provider currently supports [Gardener](https://gardener.cloud/) based container orchestration engine.

- [Terraform Provider Cleura](#terraform-provider-cleura)
  - [Prerequisites](#prerequisites)
  - [Supported platforms](#supported-platforms)
  - [Configuring the aztekas/cleura provider](#configuring-the-aztekascleura-provider)
  - [Using the aztekas/cleura provider](#using-the-aztekascleura-provider)
  - [Cleura CLI](#cleura-cli)
  - [Dependencies](#dependencies)
  - [Local development](#local-development)

## Prerequisites

- Cleura Account / API Token
- Terraform CLI version 1.0 and later

## Supported platforms

Provider supports the following platforms/architectures:

- Linux / AMD64
- Darwin / AMD64
- Darwin / ARM64
- Windows / AMD64

## Configuring the aztekas/cleura provider

To set up the provider:

1. Get Cleura API Token, either via Cleura API [directly](https://apidoc.cleura.cloud/#api-Authentication-CreateToken), or via [Cleura CLI](https://github.com/aztekas/cleura-client-go) `cleura token get` command
1. Configure provider by either:
   - Setting `username`, `token`, `host` variables in the provider configuration block.
   - Setting up corresponding environment variables (CLEURA_API_TOKEN, CLEURA_API_USERNAME, CLEURA_API_HOST)
   - Setting `config_file` variable in the provider configuration block to a configuration file path. Check `cleura config generate-template` for configuration file template.
1. Resulting configuration should look like this:

```hcl
// provider.tf

terraform {
  required_providers {
    cleura = {
      source  = "aztekas/cleura"
      version = "0.0.4"
    }
  }
}

provider "cleura" {
/* Configuration via variables
   host     = "https://rest.cleura.cloud"
   username = "your-username"
   token    = "token"
*/
/* Configuration via config file
   config_file = "/home/user/.config/cleura/config"
*/
/* Leave blank if environment variables are used
*/
}
```

Configuration file example:

```config
active_profile: default
profiles:
  default:
    username: your-username-here
    token: your-token-here
    api-url: https://rest.cleura.cloud
```

> [!NOTE]
> Generated tokens are short-lived tokens and will require re-generation once expired.

> [!WARNING]
> Configuration file stores token in open text

## Using the aztekas/cleura provider

Please check [`/examples`](./examples/) folder for more usage examples.
Basic example:

```hcl
resource "cleura_shoot_cluster" "test_cluster" {

  project = "project-id"
  region = "sto2"
  name = "test-cluster"
  kubernetes_version = "1.29.4"
  provider_details = {
    worker_groups = [
     {
        worker_group_name = "wr001"
        machine_type = "b.2c4gb"
        min_nodes = 2
        max_nodes = 3
        image_version = "1443.2.0"
      }
    ]
  }

}
```

## Cleura CLI

- Check latest cli version: <https://github.com/aztekas/cleura-client-go/releases>

## Dependencies

- Cleura API Go Client (<https://github.com/aztekas/cleura-client-go>)

## Local development

Please refer to a local development setup [docs](./LOCAL_DEVELOPMENT.md)
