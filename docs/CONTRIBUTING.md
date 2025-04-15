# Contributing to go-tfe

If you find an issue with this package, please create an issue in GitHub. If you'd like, we welcome any contributions. Fork this repository and submit a pull request.

## Adding new functionality or fixing relevant bugs

If you are adding a new endpoint, make sure to update the [coverage list in README.md](../README.md#API-Coverage) where we keep a list of the HCP Terraform APIs that this SDK supports.

If you are making relevant changes that is worth communicating to our users, please include a note about it in our CHANGELOG.md. You can include it as part of the PR where you are submitting your changes.

CHANGELOG.md should have the next minor version listed as `# v1.X.0 (Unreleased)` and any changes can go under there. But if you feel that your changes are better suited for a patch version (like a critical bug fix), you may list a new section for this version. You should repeat the same formatting style introduced by previous versions.

### Scoping pull requests that add new resources

There are instances where several new resources being added (i.e Workspace Run Tasks and Organization Run Tasks) are coalesced into one PR. In order to keep the review process as efficient and least error prone as possible, we ask that you please scope each PR to an individual resource even if the multiple resources you're adding share similarities. If joining multiple related PRs into one single PR makes more sense logistically, we'd ask that you organize your commit history by resource. A general convention for this repository is one commit for the implementation of the resource's methods, one for the integration test, and one for cleanup and housekeeping (e.g modifying the changelog/docs, generating mocks, etc).

**Note HashiCorp Employees Only:** When submitting a new set of endpoints please ensure that one of your respective team members approves the changes as well before merging.

## Linting

After opening a PR, our CI system will perform a series of code checks, one of which is linting. Linting is not strictly required for a change to be merged, but it helps smooth the review process and catch common mistakes early. If you'd like to run the linters manually, follow these steps:

1. Ensure you have [installed golangci-lint](https://golangci-lint.run/welcome/install/#local-installation)
2. Format your code by running `make fmt`
3. Run lint checks using `make lint`

## Editor Settings

We've included VSCode settings to assist with configuring the go extension. For other editors that integrate with the [Go Language Server](https://github.com/golang/tools/tree/master/gopls), the main thing to do is to add the `integration` build tags so that the test files are found by the language server. See `.vscode/settings.json` for more details.

## Generating Mocks
Ensure you have installed the [mockgen](https://github.com/uber-go/mock) tool.

You'll need to generate mocks if an existing endpoint method is modified or a new method is added. To generate mocks, simply run `./generate_mocks.sh`.

If you're adding a new API resource to go-tfe, you'll need to add a new command to `generate_mocks.sh`. For example if someone creates `example_resource.go`, you'll add:

```
mockgen -source=example_resource.go -destination=mocks/example_resource_mocks.go -package=mocks
```

You can also use the Makefile target `mocks` to add the new command:

```
FILENAME=example_resource.go make mocks
```

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
