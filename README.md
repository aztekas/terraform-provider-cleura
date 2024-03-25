# Terraform Provider Cleura (Terraform Plugin Framework)

Built on <https://github.com/hashicorp/terraform-provider-scaffolding-framework>
template repository.

- [Terraform Provider Cleura (Terraform Plugin Framework)](#terraform-provider-cleura-terraform-plugin-framework)
  - [Requirements](#requirements)
  - [Developing the Provider](#developing-the-provider)
    - [Installing](#installing)
    - [Testing local provider version](#testing-local-provider-version)
    - [Running automated tests](#running-automated-tests)
  - [Using the published version of the provider](#using-the-published-version-of-the-provider)
  - [Getting CLEURA\_API\_TOKEN](#getting-cleura_api_token)

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.20
- [Cleura Account](https://cleura.cloud/)
- [Cleura User Token](https://apidoc.cleura.cloud/#api-Authentication-CreateToken)

## Developing the Provider

### Installing

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

> Run `go env GOPATH` to find our $GOPATH

To generate or update documentation, run `go generate`.

### Testing local provider version

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

## Using the published version of the provider

The `accelerate-at-iver/cleura` provider is published in the *private* terraform registry on the Terraform cloud. To use it, you would need to add the following snippet to the `~/.terrformrc` file:

```conf
credentials "app.terraform.io" {
  # valid user API token:
  token =<USER_TOKEN>
}
```

where `USER_TOKEN` is the token generated in terraform cloud. Refer to the [official documentation](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/users#api-tokens)

> Make sure that you comment out the `provider_installation{}` section if you are switching from development mode.

## Getting CLEURA_API_TOKEN

Instead of curling Cleura API directly to get user token, you could try a helper cli tool. Install via :

```shell
go install github.com/aztekas/cleura-client-go/cmd/cleura@v0.0.1
```

To get token use `cleura token get` command. It supports several methods to supply your Cleura credentials:

```shell
❯ cl token get -h
NAME:
   cleura token get - Receive token from Cleura API using username and password                                                                                                                                                                 USAGE:
   cleura token get [command options] [arguments...]                                                                                                                                                                                            DESCRIPTION:
   Receive token from Cleura API using username and password

OPTIONS:
   --username value, -u value      Username for token request [$CLEURA_API_USERNAME]
   --password value, -p value      Password for token request. [$CLEURA_API_PASSWORD]
   --api-host value, --host value  Cleura API host (default: "https://rest.cleura.cloud") [$CLEURA_API_HOST]
   --update-config                 Save token to active configuration. NB: token saved in open text (default: false)
   --config-path value             Path to configuration file. $HOME/.config/cleura/config if not set
   --interactive, -i               Interactive mode. Input username and password in interactive mode (default: false)
   --two-factor, --2fa             Set this flag if two-factor authentication (sms) is enabled in your cleura profile  (default: false)
   --help, -h                      show help
```

On successful authentication you will get an output in the following format:

```shell
export CLEURA_API_TOKEN=<GENERATED TOKEN>
export CLEURA_API_USERNAME=<YOUR EMAIL>
export CLEURA_API_HOST=https://rest.cleura.cloud
```

Check latest cli version: <https://github.com/aztekas/cleura-client-go>
