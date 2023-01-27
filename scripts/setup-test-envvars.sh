#!/usr/bin/env bash -e
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


# setup-test-envvars.sh
#
# A helper script that uses envchain (https://github.com/sorah/envchain) to set environment variables for tests.
# It's required to have TFE_ADDRESS and TFE_TOKEN set, the others are optional.
#
type envchain >/dev/null 2>&1 || { echo >&2 "Required executable 'envchain' not found - install it via 'brew install envchain'. Exiting."; exit 1; }

echo "Set environment variables (envvars) for running tests locally"
echo " envchain will prompt you for values for 4 envvars"
echo " TFE_ADDRESS and TFE_TOKEN are required, the others are optional,"
echo " press 'return' to skip them"
echo ""

read -p "Enter the namespace you want to use in envchain [go-tfe]: " namespace
namespace=${namespace:-go-tfe}
envchain --set ${namespace} TFE_ADDRESS TFE_TOKEN OAUTH_CLIENT_GITHUB_TOKEN GITHUB_POLICY_SET_IDENTIFIER
echo "Done! To see the values: envchain ${namespace} env"
