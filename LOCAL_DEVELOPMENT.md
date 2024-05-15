# Local Development

Built on <https://github.com/hashicorp/terraform-provider-scaffolding-framework>
template repository.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21
- [Cleura Account](https://cleura.cloud/)
- [Cleura User Token](https://apidoc.cleura.cloud/#api-Authentication-CreateToken)

## Installing

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

> Run `go env GOPATH` to find our $GOPATH

To generate or update documentation, run `go generate`.

## Running automated tests

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```

Check test configuration code under [internal/provider/testdata/*/*](./internal/provider/testdata/)

## Using development version of the provider

In order to create resources with `dev` version of cleura provider, you have to setup terraform to run local version of the provider:

1. Create/modify `~/.terraformrc` file with the following content:

   ```txt
    provider_installation {

        dev_overrides {
            "registry.terraform.io/aztekas/cleura" = "$GOPATH/bin" # set path to provider binary here
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
                source = "registry.terraform.io/aztekas/cleura"
            }
        }
    }

    provider "cleura" {
        host     = "https://rest.cleura.cloud" # CLEURA_API_HOST
        username = "<username>" # CLEURA_API_USERNAME
        token = "<token>" # CLEURA_API_TOKEN
    }

   ```

1. Refer to example declarations [under examples folder](./examples)
