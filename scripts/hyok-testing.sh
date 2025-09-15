#!/bin/bash

env="STAGING_ENVCHAIN"
pairs=(
    # HYOK Attributes testing
    # -- Agent Pools
    "TestAgentPoolsRead:read_hyok_configurations_of_an_agent_pool"
    # -- Plans
    "TestPlansRead:read_hyok_encrypted_data_key_of_a_plan"
    "TestPlansRead:read_sanitized_plan_of_a_plan"
    # -- Workspaces
    "TestWorkspacesCreate:create_workspace_with_hyok_enabled_set_to_false"
    "TestWorkspacesCreate:create_workspace_with_hyok_enabled_set_to_true"
    "TestWorkspacesRead:read_hyok_enabled_of_a_workspace"
    "TestWorkspacesRead:read_hyok_encrypted_data_key_of_a_workspace"
    "TestWorkspacesUpdate:update_hyok_enabled_of_a_workspace_from_false_to_false"
    "TestWorkspacesUpdate:update_hyok_enabled_of_a_workspace_from_false_to_true"
    "TestWorkspacesUpdate:update_hyok_enabled_of_a_workspace_from_true_to_true"
    "TestWorkspacesUpdate:update_hyok_enabled_of_a_workspace_from_true_to_false"
    # -- Organizations
    "TestOrganizationsRead:read_primary_hyok_configuration_of_an_organization"
    "TestOrganizationsRead:read_enforce_hyok_of_an_organization"
    "TestOrganizationsUpdate:update_enforce_hyok_of_an_organization_to_true"
    "TestOrganizationsUpdate:update_enforce_hyok_of_an_organization_to_false"
    # -- State Versions
    "TestStateVersionsRead:read_encrypted_state_download_url_of_a_state_version"
    "TestStateVersionsRead:read_sanitized_state_download_url_of_a_state_version"
    "TestStateVersionsRead:read_hyok_encrypted_data_key_of_a_state_version"
    "TestStateVersionsUpload:uploading_state_using_SanitizedStateUploadURL_and_verifying_SanitizedStateDownloadURL_exists"
    "TestStateVersionsUpload:SanitizedStateUploadURL_is_required_when_uploading_sanitized_state"

    # AWS OIDC Configuration testing
    "TestAWSOIDCConfigurationCreateDelete:with_valid_options"
    "TestAWSOIDCConfigurationCreateDelete:missing_role_ARN"
    "TestAWSOIDCConfigurationRead:fetch_existing_configuration"
    "TestAWSOIDCConfigurationRead:fetching_non-existing_configuration"
    "TestAWSOIDCConfigurationsUpdate:with_valid_options"
    "TestAWSOIDCConfigurationsUpdate:missing_role_ARN"

    # Azure OIDC Configuration testing
    "TestAzureOIDCConfigurationCreateDelete:with_valid_options"
    "TestAzureOIDCConfigurationCreateDelete:missing_client_ID"
    "TestAzureOIDCConfigurationCreateDelete:missing_subscription_ID"
    "TestAzureOIDCConfigurationCreateDelete:missing_tenant_ID"
    "TestAzureOIDCConfigurationRead:fetch_existing_configuration"
    "TestAzureOIDCConfigurationRead:fetching_non-existing_configuration"
    "TestAzureOIDCConfigurationUpdate:update_all_fields"
    "TestAzureOIDCConfigurationUpdate:client_ID_not_provided"
    "TestAzureOIDCConfigurationUpdate:subscription_ID_not_provided"
    "TestAzureOIDCConfigurationUpdate:tenant_ID_not_provided"

    # GCP OIDC Configuration testing
    "TestGCPOIDCConfigurationCreateDelete:with_valid_options"
    "TestGCPOIDCConfigurationCreateDelete:missing_workload_provider_name"
    "TestGCPOIDCConfigurationCreateDelete:missing_service_account_email"
    "TestGCPOIDCConfigurationCreateDelete:missing_project_number"
    "TestGCPOIDCConfigurationRead:fetch_existing_configuration"
    "TestGCPOIDCConfigurationRead:fetching_non-existing_configuration"
    "TestGCPOIDCConfigurationUpdate:update_all_fields"
    "TestGCPOIDCConfigurationUpdate:workload_provider_name_not_provided"
    "TestGCPOIDCConfigurationUpdate:service_account_email_not_provided"
    "TestGCPOIDCConfigurationUpdate:project_number_not_provided"

    # Vault OIDC Configuration testing
    "TestVaultOIDCConfigurationCreateDelete:with_valid_options"
    "TestVaultOIDCConfigurationCreateDelete:missing_address"
    "TestVaultOIDCConfigurationCreateDelete:missing_role_name"
    "TestVaultOIDCConfigurationRead:fetch_existing_configuration"
    "TestVaultOIDCConfigurationRead:fetching_non-existing_configuration"
    "TestVaultOIDCConfigurationUpdate:update_all_fields"
    "TestVaultOIDCConfigurationUpdate:address_not_provided"
    "TestVaultOIDCConfigurationUpdate:role_name_not_provided"
    "TestVaultOIDCConfigurationUpdate:namespace_not_provided"
    "TestVaultOIDCConfigurationUpdate:JWTAuthPath_not_provided"
    "TestVaultOIDCConfigurationUpdate:TLSCACertificate_not_provided"

    # HYOK Customer Key Version testing
    "TestHYOKCustomerKeyVersionsList:with_no_list_options"
    "TestHYOKCustomerKeyVersionsRead:read_an_existing_key_version"

    # HYOK Encrypted Data Key testing
    "TestHYOKEncryptedDataKeyRead:read_an_existing_encrypted_data_key"

    # HYOK Configurations testing
    "TestHYOKConfigurationCreateRevokeDelete:AWS_with_valid_options"
    "TestHYOKConfigurationCreateRevokeDelete:AWS_with_missing_key_region"
    "TestHYOKConfigurationCreateRevokeDelete:GCP_with_valid_options"
    "TestHYOKConfigurationCreateRevokeDelete:GCP_with_missing_key_location"
    "TestHYOKConfigurationCreateRevokeDelete:GCP_with_missing_key_ring_ID"
    "TestHYOKConfigurationCreateRevokeDelete:Vault_with_valid_options"
    "TestHYOKConfigurationCreateRevokeDelete:Azure_with_valid_options"
    "TestHYOKConfigurationCreateRevokeDelete:with_missing_KEK_ID"
    "TestHYOKConfigurationCreateRevokeDelete:with_missing_agent_pool"
    "TestHYOKConfigurationCreateRevokeDelete:with_missing_OIDC_config"
    "TestHyokConfigurationList:without_list_options"
    "TestHyokConfigurationRead:AWS"
    "TestHyokConfigurationRead:Azure"
    "TestHyokConfigurationRead:GCP"
    "TestHyokConfigurationRead:Vault"
    "TestHyokConfigurationRead:fetching_non-existing_configuration"
    "TestHYOKConfigurationUpdate:AWS_with_valid_options"
    "TestHYOKConfigurationUpdate:GCP_with_valid_options"
    "TestHYOKConfigurationUpdate:Vault_with_valid_options"
    "TestHYOKConfigurationUpdate:Azure_with_valid_options"
)

for pair in "${pairs[@]}"; do
    IFS=':' read -r parent child <<< "$pair"
    result=$(envchain ${env} go test -run "^${parent}$/^${child}$" -v ./...)
    status="\033[33mUNKNOWN\033[0m" # yellow by default
    if echo "$result" | grep -q "^    --- SKIP: ${parent}/${child}"; then
        status="\033[33mSKIP\033[0m" # yellow
    elif echo "$result" | grep -q "^--- PASS: ${parent}"; then
        status="\033[32mPASS\033[0m" # green
    elif echo "$result" | grep -q "^--- FAIL: ${parent}"; then
        status="\033[31mFAIL\033[0m" # red
    fi
    echo -e "\033[34m${parent}/${child}\033[0m: ${status}"
done