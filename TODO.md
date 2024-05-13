# TODO and BUGS

## TODO

1. [ ] Add maintenance window functionality
1. [ ] Test out timeout functionality
1. [ ] Add support for reading token from the cleura config file (as with cleura cli)

## DELAYED

1. [ ] Consider state move and sate upgrade functionality. Should be implemented when needed (use case: parameter deprecation, new parameters in existing datasources and/or resources)

## BUGS or FEATURES

1. Got 409 error when updating `image_version` on all worker groups simultaneously. Not repeatable error.
1. (API) If adding several hibernation schedules it is not possible to remove one from the list. API errors with internal error, same behavior via UI console.
1. Somehow waiter functionality fails (clusterReadyOperationWaiter), and cluster can be shown as created right after `terraform apply` is run. (not repeatable)

## IMPLEMENTED/FIXED

1. [x] Allow possibility to omit specification of worker group name.
1. [x] Check how/if timeout for `create` and `delete` work.
1. [x] Add APIError struct to parse errors from Cleura API
1. [x] Check/read what happens when user presses Ctrl+c during the ongoing operation create/delete. Operation is not reverted.
1. [x] Implement Update operation
1. [x] Clean up the repo (scaffolding stuff)
1. [x] Set Github Actions workflow for building provider
1. [x] Publish provider somewhere or document local usage
1. [x] Move Cleura go client to a separate repository
1. [x] Add basic testing
1. [x] Add possibility to provide token string to provider configuration. Will require getting the correct token outside terraform.
1. [x] Add Import functionality (for moving existing resources into terraform state)
1. [x] Do not allow empty worker_groups list, same way as for hibernation schedules (via list validator)
1. [x] Add datasource for projects. Openstack provider can be used here.
1. [x] (docs) Add description fields to the shoot cluster resource schema.
1. [x] (docs) Add description fields to the shoot cluster datasource schema.
1. [x] (docs) Add description fields to the provider schema
1. [x] If cluster in destroyed via ui (not reflected in state) then terraform should update the state and try to create a new one with the given parameters defined in `cleura_shoot_cluster` resource. At the moment it outputs "Error Reading Shoot cluster".
1. [x] Calibrate exponential backoff. Start non random status checks every 30s after ~6.5 minutes.
