#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


if [[ -z $1 ]]; then
    echo "Please specify the pull request number you want to rebase to a local branch."
    echo "Usage: ./scripts/rebase-fork.sh <pr number>"
    echo ""
    echo "Example: ./scripts/rebase-fork.sh 557"
    exit 1
fi

PR_NUMBER=$1

declare -a req_tools=("gh" "git" "jq")
for tool in "${req_tools[@]}"; do
  if ! command -v "${tool}" > /dev/null; then
    echo "It looks like '${tool}' is not installed; please install it and run this script again."
    exit 1
  fi
done

# Check if the PR specified is a valid number
re='^[0-9]+$'
if ! [[ ${PR_NUMBER} =~ ${re} ]] ; then
   echo "The PR you specify must be a valid integer number." >&2; exit 1
fi

# Check if the specified PR exists
# We only capture stderr here and redirect stdout to /dev/null
errormsg=$(gh pr view ${PR_NUMBER} 2>&1 1>/dev/null)
if [[ ! -z ${errormsg} ]]; then
    # strip GraphQL log prefix to keep the error message clean
    errormsg=${errormsg#"GraphQL: "}
    echo "Failed to fetch pull request #${PR_NUMBER}: ${errormsg}"
    exit 1
fi

# Check if the pull request we want to rebase is already closed. If so
# exit.
closed=$(gh pr view ${PR_NUMBER} --json closed | jq '.closed')
if [[ $closed = "true" ]]; then
    echo "The pull request #${PR_NUMBER} to rebase is already closed."
    exit 1
fi

# Save the name of the current branch we're in so we can go back to it after
# we are done
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

# Checkout the fork PR locally
gh pr checkout ${PR_NUMBER}

# Grab the PR title and branch name
FORK_PR_TITLE=$(gh pr view ${PR_NUMBER} --json title | jq '.title' | tr -d '"')
FORK_PR_BRANCH=$(gh pr view ${PR_NUMBER} --json headRefName | jq '.headRefName')

# Fetch the username of the user currently authenticated with gh cli
USER=$(gh api user | jq -r '.login')

# Name of the local branch that will be pushed upstream
LOCAL_BRANCH="${USER}/$(echo ${FORK_PR_BRANCH} | tr -d '"')"

# Fetch the PR body and write to local markdown file
gh pr view ${PR_NUMBER} --json body | jq -r '.body' > ${PR_NUMBER}.md

git checkout -b ${LOCAL_BRANCH}
git commit --allow-empty -m "Rebased ${FORK_PR_BRANCH} onto a local branch"
git push -u origin ${LOCAL_BRANCH}

# Finally we can automagically open a new PR using the fork PR's original title
# and description
gh pr create --title="${FORK_PR_TITLE}" --body-file=${PR_NUMBER}.md

# Cleanup
rm ${PR_NUMBER}.md
git checkout ${CURRENT_BRANCH}

