## Example: Parsing Run Errors

In this example, you'll use terraform to create a run with errors on Terraform Cloud, then
execute the command to read the plan log and filter it for errors. It's important to use
Terraform to create the run, otherwise you will not get the structured log that this code
example requires.

#### Instructions

1. Change to the terraform directory, and run terraform init using Terraform 1.3+

`cd terraform`
`TF_CLOUD_ORGANIZATION="yourorg" terraform init`

2. Apply the changes (You should see an error "Error making request" or similar)

`TF_CLOUD_ORGANIZATION="yourorg" terraform apply`

3. Notice the run ID in the URL (it begins with "run-") and execute the example with the run ID as a flag:

`cd ../`
`TFE_TOKEN="YOURTOKEN" go run main.go run-RUN_ID_FROM_URL_ABOVE`
