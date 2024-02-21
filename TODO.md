# TODO and BUGS

## TODO

1. [ ] If cluster in destroyed via ui (not reflected in state) then terraform should update the state and try to create a new one with the given parameters defined in `cleura_shoot_cluster` resource. At the moment it outputs "Error Reading Shoot cluster". Do not remember how it works in,say, google provider.
1. [x] Allow possibility to omit specification of worker group name.
1. [x] Check how/if timeout for `create` and `delete` work.
1. [x] Add APIError struct to parse errors from Cleura API
1. [ ] Check/read what happens when user presses Ctrl+c during the ongoing operation create/delete.
1. [x] Implement Update operation
1. [x] Clean up the repo (scaffolding stuff)
1. [ ] Set Github Actions workflow for building provider
1. [ ] Publish provider somewhere or document local usage
1. [ ] Move Cleura go client to a separate repository
1. [ ] Add testing
1. [ ] Change of all fields except presented here: <https://apidoc.cleura.cloud/#api-Gardener-UpdateShoot>  must lead to cluster re-creation
1. [ ] Add token revoke call before terraform is finished running command
1. [ ] Add possibility to provide token string to provider configuration. Will require getting the correct token outside terraform.

## BUGS or FEATURES

1. [ ] Got 409 error when updating `image_version` on all worker groups simultaneously. Not repeatable error.
