# Contributing to go-tfe

If you find an issue with this package, please create an issue in GitHub. If you'd like, we welcome any contributions. Fork this repository and submit a pull request.

### Writing Tests

The test suite contains many acceptance tests that are run against the latest version of Terraform Enterprise. You can read more about running the tests against your own Terraform Enterprise environment in [TESTS.md](TESTS.md). Our CI system (Circle) will not test your fork unless you are an authorized employee, so a HashiCorp maintainer will initiate the tests or you and report any missing tests or simple problems. In order to speed up this process, it's not uncommon for your commits to be incorportated into another PR that we can commit test changes to.

### Editor Settings

We've included VSCode settings to assist with configuring the go extension. For other editors that integrate with the [Go Language Server](https://github.com/golang/tools/tree/master/gopls), the main thing to do is to add the `integration` build tags so that the test files are found by the language server. See `.vscode/settings.json` for more details.
