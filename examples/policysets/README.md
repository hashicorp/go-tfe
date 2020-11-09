# Policy Sets Example
This example illustrates how to create a Sentinel policy set and a policy set version. It also shows how to upload a policy set containing a policy set definition file and an actual policy to the upload URL of the policy set version. These files are loaded from this [directory](../../test-fixtures/policy-set-version).

## Installation
Run `go install .`

## Running
Export the `TFE_ADDRESS`, `TFE_TOKEN`, and `TFE_ORGANIZATION` environment variables. Be sure to specify an organization that has a pricing plan that supports Sentinel policy sets.

Then run `$GOPATH/bin/policysets`.

You should see output like this:
```
2020/11/09 10:35:46 The policy set ID is:polset-vqzQART8KsoW3Yjq
2020/11/09 10:35:46 The policy set version Type is: policy-set-versions
2020/11/09 10:35:46 The policy set version ID is: polsetver-yjV8gjDNAZm7JJyN
2020/11/09 10:35:46 The linked policy set ID is: polset-vqzQART8KsoW3Yjq
2020/11/09 10:35:46 The upload link is:https://tfe.hashidemos.io/_archivist/v1/object/dmF1bHQ6djE6bHoyN2RUTFFSVEFSOFhUT0l1UElQUmZLN20wTENxWVRIcUdNMXRxNzhKTDJnWCtPcjVhWFdTRS8wSE9jdTYxWTArejlXOEhXY1J1bWVkclp6bUdwVW1wTkRXUEJyU0VpMVllZ1ozVHQ5cmgveFhMS25BT09vcC9wcy9MVWRCQjI1SXcrZUNtUmdQRU9Oc0JsWm9MN2ZqSmhpU2FMbng0UWI2NU1nYloreTBiclJpdmlJMzJYckNmU2NwRXNIQVB6VGs4WFMwWWY5NS9vYjlJcUZXUklpa0F6alRlZTl3bkd4b1VDRnpqSElhQkZvWE5tNmFYaFh5NXpGM095ckFockk3d2w2TldWWHRaMzFTdXE5b0xJUzkrUA
2020/11/09 10:35:46 The upload status is: pending
2020/11/09 10:35:46 The upload URL is:https://tfe.hashidemos.io/_archivist/v1/object/dmF1bHQ6djE6bHoyN2RUTFFSVEFSOFhUT0l1UElQUmZLN20wTENxWVRIcUdNMXRxNzhKTDJnWCtPcjVhWFdTRS8wSE9jdTYxWTArejlXOEhXY1J1bWVkclp6bUdwVW1wTkRXUEJyU0VpMVllZ1ozVHQ5cmgveFhMS25BT09vcC9wcy9MVWRCQjI1SXcrZUNtUmdQRU9Oc0JsWm9MN2ZqSmhpU2FMbng0UWI2NU1nYloreTBiclJpdmlJMzJYckNmU2NwRXNIQVB6VGs4WFMwWWY5NS9vYjlJcUZXUklpa0F6alRlZTl3bkd4b1VDRnpqSElhQkZvWE5tNmFYaFh5NXpGM095ckFockk3d2w2TldWWHRaMzFTdXE5b0xJUzkrUA
2020/11/09 10:35:46 The upload status is: ready
2020/11/09 10:35:46 Successfully deleted policy set polset-vqzQART8KsoW3Yjq
```
