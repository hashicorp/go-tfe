## Release Process

go-tfe can be released as often as required. Documentation updates and test fixes that only touch test files don't require a release or tag. You can just merge these changes into `main` once they have been approved.

### Preparing a release

Start by comparing the main branch with the last release in order to fully understand which changes are being released. Compare the last release tag with main ([example](https://github.com/hashicorp/go-tfe/compare/v1.5.0...main)). For each meaningful change, double check the following:

1. Is the change added to CHANGELOG.md?
2. Does the public package API follow all endpoint conventions, such as naming, pointer usage, and options availability? Once these are released, they are permanent in the current major release version.
3. Are new features generally available in the Terraform Cloud API? Or is there another considered reason to release them?

Steps to prepare the changelog for a new release:

1. Replace `# Unreleased` with the version you are releasing.
2. Ensure there is a line with `# Unreleased` at the top of the changelog for future changes. Ideally we don't ask authors to add this line; this will make it clear where they should add their changelog entry.

Open a pull request with these changes titled `vX.XX.XX Changelog`. Once approved and merged, you can go ahead and create the release.

### Creating a release

1. [Create a new release in GitHub](https://help.github.com/en/github/administering-a-repository/creating-releases) by clicking on "Releases" and then "Draft a new release"
2. Set the `Tag version` to a new tag, using [Semantic Versioning](https://semver.org/) as a guideline.
3. Set the `Target` as `main`.
4. Set the `Release title` to the tag you created, `vX.Y.Z`
5. Use the description section to describe why you're releasing and what changes you've made. You should include links to merged PRs. Use the following headers in the description of your release:
   - BREAKING CHANGES: Use this for any changes that aren't backwards compatible. Include details on how to handle these changes.
   - FEATURES: Use this for any large new features added,
   - ENHANCEMENTS: Use this for smaller new features added
   - BUG FIXES: Use this for any bugs that were fixed.
   - NOTES: Use this section if you need to include any additional notes on things like upgrading, upcoming deprecations, or any other information you might want to highlight.

   Markdown example:

   ```markdown
   ENHANCEMENTS
   * Add description of new small feature (#3)[link-to-pull-request]

   BUG FIXES
   * Fix description of a bug (#2)[link-to-pull-request]
   * Fix description of another bug (#1)[link-to-pull-request]
   ```

6. Don't attach any binaries. The zip and tar.gz assets are automatically created and attached after you publish your release.
7. Click "Publish release" to save and publish your release.
