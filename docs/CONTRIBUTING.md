# Contributing to go-tfe

### Adding or Updating HCP Terraform v2 API

To add functionality to go-tfe/v2, edit the OpenAPI specification in the Terraform Platform code.
The github.com/hashicorp/go-tfe/v2 package will be generated from that specification nightly and
will include the new functionality. Please note that endpoints and attributes labelled as
"public-beta" will appear in this client.

### v1 package (root directory) Contributions

v1.go contains the final version of the go-tfe (v1) package. You may add critical fixes or security
updates to v1.go, but the functionality is NO LONGER TESTED and SHOULD NOT BE EXTENDED except for
in uncommon situations as determined by @hashicorp/tf-core-cloud.

### go-tfe Core Contributions

Everything outside of the `v2/api` directory is maintained as the core go-tfe wrapper code and
handles the following features and functionality:

- Configuration
- Authentication
- Meta APIs (IPRanges and OpenAPI)
- Decompression
- Automatic retries
- Error handling
- Streaming downloads by URL or path (undecoded bodies)