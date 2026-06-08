# Contributing to go-tfe

If you find an issue with this package, please create an issue in GitHub. If you'd like, we welcome any contributions. Fork this repository and submit a pull request.

## Linting

After opening a PR, our CI system will perform a series of code checks, one of which is linting. Linting is not strictly required for a change to be merged, but it helps smooth the review process and catch common mistakes early. If you'd like to run the linters manually, follow these steps:

1. Ensure you have [installed golangci-lint](https://golangci-lint.run/welcome/install/#local-installation)
2. Format your code by running `make fmt`
3. Run lint checks using `make lint`

## Editor Settings

We've included VSCode settings to assist with configuring the go extension. For other editors that integrate with the [Go Language Server](https://github.com/golang/tools/tree/master/gopls), the main thing to do is to add the `integration` build tags so that the test files are found by the language server. See `.vscode/settings.json` for more details.

## Rebasing a fork to trigger CI (Maintainers Only)

Pull requests that originate from a fork will not have access to this repository's secrets, thus resulting in the inability to test against our CI instance. In order to trigger the CI action workflow, there is a handy script `./scripts/rebase-fork.sh` that automates the steps for you. It will:

* Checkout the fork PR locally onto your machine and create a new branch prefixed as follows: `local/{name_of_fork_branch}`
* Push your newly created branch to Github, appending an empty commit stating the original branch that was rebased.
* Copy the contents of the fork's pull request (title and description) and create a new pull request, triggering the CI workflow.

**Important**: This script does not handle subsequent commits to the original PR and would require you to rebase them manually. Therefore, it is important that authors include test results in their description and changes are approved before this script is executed.

This script depends on `gh` and `jq`. It also requires you to `gh auth login`, providing a SSO-authorized personal access token with the following scopes enabled:

- repo
- read:org
- read:discussion

### Example Usage

```sh
./scripts/rebase-fork.sh 557
```
