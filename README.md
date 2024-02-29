# Terraform Provider Cleura (Terraform Plugin Framework)

Built on <https://github.com/hashicorp/terraform-provider-scaffolding-framework>
template repository.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.20

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Developing the Provider

### Installing

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

### Testing provider

In order to create resources with `dev` version of cleura provider, you have to setup terraform to run local version of the provider:

1. Create/modify `~/.terraformrc` file with the following content:

   ```txt
    provider_installation {

        dev_overrides {
            "app.terraform.io/accelerate-at-iver/cleura" = "$GOPATH/bin" # set path to provider binary here
        }

        # For all other providers, install them directly from their origin provider
        # registries as normal. If you omit this, Terraform will _only_ use
        # the dev_overrides block, and so no other providers will be available.
        direct {}

    }

   ```

1. Configure provider:

   ```hcl
    #provider.tf

    terraform {
        required_providers {
            cleura = {
                source = "app.terraform.io/accelerate-at-iver/cleura"
            }
        }
    }

    provider "cleura" {
        host     = "https://rest.cleura.cloud" # CLEURA_API_HOST
        username = "<username>" # CLEURA_API_USERNAME
        token = "<token>" # CLEURA_API_TOKEN
    }

   ```

1. Refer to example resource declaration [under examples folder](./examples/resources/)

### Running automated tests

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```

Check test configuration code under [internal/provider/testdata/*/*](./internal/provider/testdata/)
